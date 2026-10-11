package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

type GameTitleAccess struct {
	PlayerUUID      string   `json:"playerUuid"`
	SessionEpoch    string   `json:"sessionEpoch"`
	CatalogRevision int64    `json:"catalogRevision"`
	AllowedIDs      []string `json:"allowedIds"`
	TitleID         string   `json:"titleId,omitempty"`
	ClientRequestID string   `json:"clientRequestId,omitempty"`
	ExpectedVersion int64    `json:"expectedVersion,omitempty"`
}
type GameTitlePlayer struct {
	PlayerUUID  string `json:"playerUuid"`
	GameID      string `json:"gameId"`
	TitleID     string `json:"titleId"`
	DisplayName string `json:"displayName"`
	Version     int64  `json:"version"`
}

// Access is supplied only by authenticated Core, never by the player client.
func (s *Store) ReconcileGameTitle(ctx context.Context, name string, in GameTitleAccess) (GameTitlePlayer, error) {
	out := GameTitlePlayer{PlayerUUID: in.PlayerUUID, GameID: name}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	var rev int64
	// Lock catalog before player, consistently with every reconciliation.
	if err = tx.QueryRowContext(ctx, "SELECT revision FROM game_title_revision WHERE id=1 LOCK IN SHARE MODE").Scan(&rev); err != nil {
		return out, err
	}
	if rev != in.CatalogRevision {
		return out, ErrConflict
	}
	// Take an exclusive row lock even when it already exists; INSERT IGNORE's
	// shared duplicate-key lock can deadlock two simultaneous lock upgrades.
	if _, err = tx.ExecContext(ctx, "INSERT INTO game_title_players(player_uuid,game_id) VALUES(?,?) ON DUPLICATE KEY UPDATE player_uuid=VALUES(player_uuid)", in.PlayerUUID, name); err != nil {
		return out, err
	}
	var cursor int64
	if err = tx.QueryRowContext(ctx, "SELECT title_id,history_cursor,version FROM game_title_players WHERE player_uuid=? FOR UPDATE", in.PlayerUUID).Scan(&out.TitleID, &cursor, &out.Version); err != nil {
		return out, err
	}
	allowed := map[string]bool{DefaultTitleID: true}
	for _, id := range in.AllowedIDs {
		allowed[id] = true
	}
	definitions := map[string]string{}
	rows, err := tx.QueryContext(ctx, "SELECT id,display_name FROM game_title_catalog WHERE status='PUBLISHED'")
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var id, display string
		if err = rows.Scan(&id, &display); err != nil {
			rows.Close()
			return out, err
		}
		if allowed[id] {
			definitions[id] = display
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	original := out.TitleID
	if _, ok := definitions[out.TitleID]; !ok {
		out.TitleID = DefaultTitleID
		rows, err = tx.QueryContext(ctx, "SELECT sequence_id,title_id FROM game_title_history WHERE player_uuid=? AND sequence_id<? ORDER BY sequence_id DESC", in.PlayerUUID, cursor)
		if err != nil {
			return out, err
		}
		found := false
		for rows.Next() {
			var seq int64
			var id string
			if err = rows.Scan(&seq, &id); err != nil {
				rows.Close()
				return out, err
			}
			if _, ok := definitions[id]; ok {
				out.TitleID = id
				cursor = seq
				found = true
				break
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return out, err
		}
		if !found {
			cursor = 0
		}
	}
	changed := original != out.TitleID
	if in.TitleID != "" {
		if !ValidSocialID(in.ClientRequestID) {
			return out, ErrSocialInvalid
		}
		raw, _ := json.Marshal([]any{in.TitleID, in.ExpectedVersion})
		fingerprint := Digest(raw)
		var old string
		e := tx.QueryRowContext(ctx, "SELECT fingerprint FROM game_title_requests WHERE player_uuid=? AND request_id=?", in.PlayerUUID, in.ClientRequestID).Scan(&old)
		if e == nil {
			if old != fingerprint {
				return out, ErrConflict
			}
		} else if !errors.Is(e, sql.ErrNoRows) {
			return out, e
		} else {
			if in.ExpectedVersion != out.Version {
				return out, ErrConflict
			}
			if _, ok := definitions[in.TitleID]; !ok {
				return out, ErrUnauthorized
			}
			result, e := tx.ExecContext(ctx, "INSERT INTO game_title_history(player_uuid,title_id) VALUES(?,?)", in.PlayerUUID, in.TitleID)
			if e != nil {
				return out, e
			}
			cursor, e = result.LastInsertId()
			if e != nil {
				return out, e
			}
			out.TitleID = in.TitleID
			changed = true
			if _, e = tx.ExecContext(ctx, "INSERT INTO game_title_requests VALUES(?,?,?)", in.PlayerUUID, in.ClientRequestID, fingerprint); e != nil {
				return out, e
			}
		}
	}
	if changed {
		out.Version++
	}
	if _, err = tx.ExecContext(ctx, "UPDATE game_title_players SET game_id=?,title_id=?,history_cursor=?,version=? WHERE player_uuid=?", name, out.TitleID, cursor, out.Version, in.PlayerUUID); err != nil {
		return out, err
	}
	out.DisplayName = definitions[out.TitleID]
	return out, tx.Commit()
}
func (s *Store) GameTitleProfiles(ctx context.Context) ([]GameTitlePlayer, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT p.player_uuid,p.game_id,p.title_id,COALESCE(t.display_name,d.display_name),p.version FROM game_title_players p LEFT JOIN game_title_catalog t ON t.id=p.title_id AND t.status='PUBLISHED' JOIN game_title_catalog d ON d.id='title_default' ORDER BY p.player_uuid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []GameTitlePlayer{}
	for rows.Next() {
		var p GameTitlePlayer
		if err = rows.Scan(&p.PlayerUUID, &p.GameID, &p.TitleID, &p.DisplayName, &p.Version); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

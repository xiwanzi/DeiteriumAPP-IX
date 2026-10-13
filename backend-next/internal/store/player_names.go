package store

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"
)

var ErrNameConflict = errors.New("player name conflicts with another login identity")
var ErrNameProofStale = errors.New("newer verified player name already committed")

type PlayerName struct {
	Sequence       int64     `json:"sequence"`
	UUID           string    `json:"uuid"`
	PlayerRef      string    `json:"playerRef"`
	PreviousGameID string    `json:"previousGameId"`
	GameID         string    `json:"gameId"`
	ChangedAt      time.Time `json:"changedAt"`
}

func (s *Store) UserByUUID(ctx context.Context, uuid string) (u User, err error) {
	err = s.DB.QueryRowContext(ctx, `SELECT id,player_ref,game_id,qq,COALESCE(server_uuid,''),password_hash,status,COALESCE(email,''),game_verified,COALESCE(display_name,game_id) FROM identities WHERE server_uuid=? AND game_verified=TRUE AND status<>'deleted'`, uuid).Scan(&u.ID, &u.PlayerRef, &u.GameID, &u.QQ, &u.ServerUUID, &u.PasswordHash, &u.Status, &u.Email, &u.GameVerified, &u.DisplayName)
	u.IdentityStatus = "bound"
	return
}
func (s *Store) KnownPlayerName(ctx context.Context, uuid string) (name string, err error) {
	err = s.DB.QueryRowContext(ctx, "SELECT game_id FROM player_names_state WHERE player_uuid=?", uuid).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		err = s.DB.QueryRowContext(ctx, "SELECT game_id FROM identities WHERE server_uuid=? AND status<>'deleted'", uuid).Scan(&name)
	}
	return
}

// SyncVerifiedNames accepts only profiles already verified by the server against
// official identity sources. It never creates accounts, moves UUIDs, or revokes sessions.
// A batch permits official name-reuse chains without assigning the old owner's assets.
func (s *Store) SyncVerifiedNames(ctx context.Context, profiles []AdmissionProfile) ([]PlayerName, error) {
	return s.SyncVerifiedNamesObserved(ctx, profiles, time.Now().UTC())
}
func (s *Store) SyncVerifiedNamesObserved(ctx context.Context, profiles []AdmissionProfile, observedAt time.Time) ([]PlayerName, error) {
	if len(profiles) == 0 || len(profiles) > 16 {
		return nil, ErrSocialInvalid
	}
	profiles = append([]AdmissionProfile(nil), profiles...)
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].UUID < profiles[j].UUID })
	wanted := map[string]string{}
	seenNames := map[string]bool{}
	for _, p := range profiles {
		if strings.EqualFold(p.Name, "DIMA") || strings.EqualFold(p.Name, "DaoYu") {
			return nil, ErrNameConflict
		}
		if !ValidAdmissionUUID(p.UUID) || !ValidAdmissionName(p.Name) || wanted[p.UUID] != "" || seenNames[strings.ToLower(p.Name)] {
			return nil, ErrSocialInvalid
		}
		wanted[p.UUID] = p.Name
		seenNames[strings.ToLower(p.Name)] = true
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// Rare identity changes serialize against signup/erasure and ordinary identity writes.
	var guard int
	if err = tx.QueryRowContext(ctx, "SELECT id FROM account_lifecycle_guard WHERE id=1 FOR UPDATE").Scan(&guard); err != nil {
		return nil, err
	}
	type current struct {
		p                       AdmissionProfile
		id, ref, old, qq, email string
		verified                bool
	}
	currentProfiles := []current{}
	owners := map[string]bool{}
	for _, p := range profiles {
		deleted, e := deletedUUIDTx(ctx, tx, p.UUID)
		if e != nil {
			return nil, e
		}
		if deleted {
			return nil, ErrSocialNotFound
		}
		c := current{p: p}
		e = tx.QueryRowContext(ctx, "SELECT id,player_ref,game_id,qq,COALESCE(email,''),game_verified FROM identities WHERE server_uuid=? AND status<>'deleted' FOR UPDATE", p.UUID).Scan(&c.id, &c.ref, &c.old, &c.qq, &c.email, &c.verified)
		if e != nil && !errors.Is(e, sql.ErrNoRows) {
			return nil, e
		}
		if c.id != "" && !c.verified {
			return nil, ErrSocialInvalid
		}
		if c.id == "" {
			c.ref = "player_" + Digest([]byte("deuterium-player:" + p.UUID))[:40]
			e = tx.QueryRowContext(ctx, "SELECT player_ref,game_id FROM player_names_state WHERE player_uuid=? FOR UPDATE", p.UUID).Scan(&c.ref, &c.old)
			if e != nil && !errors.Is(e, sql.ErrNoRows) {
				return nil, e
			}
		} else {
			owners[c.id] = true
		}
		var knownName string
		var knownVersion int64
		var changedAt time.Time
		stateErr := tx.QueryRowContext(ctx, "SELECT game_id,version,observed_at FROM player_names_state WHERE player_uuid=? FOR UPDATE", p.UUID).Scan(&knownName, &knownVersion, &changedAt)
		if stateErr != nil && !errors.Is(stateErr, sql.ErrNoRows) {
			return nil, stateErr
		}
		if knownVersion > 0 && knownName != p.Name && changedAt.After(observedAt) {
			return nil, ErrNameProofStale
		}
		currentProfiles = append(currentProfiles, c)
	}
	for _, c := range currentProfiles {
		if c.id == "" {
			continue
		}
		var aliasOwner, aliasGameID, aliasQQ, aliasEmail string
		e := tx.QueryRowContext(ctx, `SELECT a.user_id,i.game_id,i.qq,COALESCE(i.email,'') FROM identity_aliases a JOIN identities i ON i.id=a.user_id WHERE a.alias_key=LOWER(?)`, c.p.Name).Scan(&aliasOwner, &aliasGameID, &aliasQQ, &aliasEmail)
		if e != nil && !errors.Is(e, sql.ErrNoRows) {
			return nil, e
		}
		if e == nil && aliasOwner != c.id && (!owners[aliasOwner] || !strings.EqualFold(aliasGameID, c.p.Name) || strings.EqualFold(aliasQQ, c.p.Name) || strings.EqualFold(aliasEmail, c.p.Name)) {
			return nil, ErrNameConflict
		}
		// Unverified signups do not silently lose their reserved login names.
		var occupied int
		rows, e := tx.QueryContext(ctx, "SELECT id FROM identities WHERE LOWER(game_id)=LOWER(?) AND id<>? AND status<>'deleted'", c.p.Name, c.id)
		if e != nil {
			return nil, e
		}
		for rows.Next() {
			var id string
			if e = rows.Scan(&id); e != nil {
				rows.Close()
				return nil, e
			}
			if !owners[id] {
				occupied++
			}
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, e
		}
		if occupied > 0 {
			return nil, ErrNameConflict
		}
	}
	// Release old game-name aliases as a batch, retaining email/QQ aliases exactly.
	for _, c := range currentProfiles {
		if c.id != "" && !strings.EqualFold(c.old, c.p.Name) && !strings.EqualFold(c.old, c.qq) && !strings.EqualFold(c.old, c.email) {
			if _, err = tx.ExecContext(ctx, "DELETE FROM identity_aliases WHERE user_id=? AND alias_key=LOWER(?)", c.id, c.old); err != nil {
				return nil, err
			}
		}
	}
	changes := []PlayerName{}
	for _, c := range currentProfiles {
		if c.id != "" {
			if _, err = tx.ExecContext(ctx, "INSERT INTO identity_aliases(alias_key,user_id) VALUES(LOWER(?),?) ON DUPLICATE KEY UPDATE alias_key=VALUES(alias_key)", c.p.Name, c.id); err != nil {
				return nil, err
			}
			var aliasOwner string
			if err = tx.QueryRowContext(ctx, "SELECT user_id FROM identity_aliases WHERE alias_key=LOWER(?)", c.p.Name).Scan(&aliasOwner); err != nil {
				return nil, err
			}
			if aliasOwner != c.id {
				return nil, ErrNameConflict
			}
			if c.old != c.p.Name {
				if _, err = tx.ExecContext(ctx, "UPDATE identities SET game_id=?,admin_version=admin_version+1,updated_at=UTC_TIMESTAMP(6) WHERE id=?", c.p.Name, c.id); err != nil {
					return nil, err
				}
			}
		}
		var previous string
		var version int64
		e := tx.QueryRowContext(ctx, "SELECT game_id,version FROM player_names_state WHERE player_uuid=? FOR UPDATE", c.p.UUID).Scan(&previous, &version)
		if e != nil && !errors.Is(e, sql.ErrNoRows) {
			return nil, e
		}
		if previous == "" {
			previous = c.old
		}
		if previous == c.p.Name && c.old == c.p.Name {
			// Even an unchanged fresh proof fences older in-flight observations.
			if _, err = tx.ExecContext(ctx, "UPDATE player_names_state SET observed_at=GREATEST(observed_at,?) WHERE player_uuid=?", observedAt.UTC(), c.p.UUID); err != nil {
				return nil, err
			}
			continue
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO player_names_state(player_uuid,player_ref,game_id,version,observed_at,updated_at) VALUES(?,?,?,?,?,UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE player_ref=VALUES(player_ref),game_id=VALUES(game_id),version=VALUES(version),observed_at=VALUES(observed_at),updated_at=VALUES(updated_at)`, c.p.UUID, c.ref, c.p.Name, version+1, observedAt.UTC()); err != nil {
			return nil, err
		}
		result, e := tx.ExecContext(ctx, "INSERT INTO player_name_changes(player_uuid,player_ref,previous_game_id,game_id) VALUES(?,?,?,?)", c.p.UUID, c.ref, previous, c.p.Name)
		if e != nil {
			return nil, e
		}
		seq, e := result.LastInsertId()
		if e != nil {
			return nil, e
		}
		for _, q := range []string{"UPDATE core_player_directory SET game_id=? WHERE player_uuid=?", "UPDATE admission_entries SET game_id=? WHERE server_uuid=?", "UPDATE game_title_players SET game_id=? WHERE player_uuid=?"} {
			if _, err = tx.ExecContext(ctx, q, c.p.Name, c.p.UUID); err != nil {
				return nil, err
			}
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES('official-name-sync','account.name-sync',?,UTC_TIMESTAMP(6))", c.ref); err != nil {
			return nil, err
		}
		changes = append(changes, PlayerName{Sequence: seq, UUID: c.p.UUID, PlayerRef: c.ref, PreviousGameID: previous, GameID: c.p.Name, ChangedAt: time.Now().UTC()})
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return changes, nil
}

func (s *Store) PlayerNameChanges(ctx context.Context, after int64) ([]PlayerName, int64, bool, error) {
	if after < 0 {
		return nil, after, false, ErrSocialInvalid
	}
	var upper int64
	if e := s.DB.QueryRowContext(ctx, "SELECT COALESCE(MAX(sequence_id),0) FROM player_name_changes").Scan(&upper); e != nil {
		return nil, after, false, e
	}
	rows, e := s.DB.QueryContext(ctx, `SELECT c.sequence_id,c.player_uuid,c.player_ref,c.previous_game_id,c.game_id,c.changed_at FROM player_name_changes c WHERE c.sequence_id>? AND c.sequence_id<=? AND NOT EXISTS(SELECT 1 FROM account_deletions d WHERE d.player_ref=c.player_ref) ORDER BY c.sequence_id LIMIT 101`, after, upper)
	if e != nil {
		return nil, after, false, e
	}
	defer rows.Close()
	out := []PlayerName{}
	for rows.Next() {
		var n PlayerName
		if e = rows.Scan(&n.Sequence, &n.UUID, &n.PlayerRef, &n.PreviousGameID, &n.GameID, &n.ChangedAt); e != nil {
			return nil, after, false, e
		}
		out = append(out, n)
	}
	if e = rows.Err(); e != nil {
		return nil, after, false, e
	}
	more := len(out) > 100
	if more {
		out = out[:100]
	}
	if len(out) > 0 {
		after = out[len(out)-1].Sequence
	}
	if !more && upper > after {
		after = upper
	}
	return out, after, more, nil
}

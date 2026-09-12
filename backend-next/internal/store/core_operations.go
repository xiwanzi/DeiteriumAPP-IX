package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"
)

type CoreOperation struct {
	ID        string          `json:"operationId"`
	ActorID   string          `json:"-"`
	NodeID    string          `json:"serverId"`
	Command   string          `json:"command"`
	Payload   json.RawMessage `json:"-"`
	State     string          `json:"status"`
	Result    json.RawMessage `json:"result,omitempty"`
	ExpiresAt time.Time       `json:"expiresAt"`
	CreatedAt time.Time       `json:"createdAt"`
}

func (s *Store) CreateCoreOperation(ctx context.Context, actor, client, node, command string, payload []byte) (op CoreOperation, replay bool, err error) {
	hash := Digest(append([]byte(node+":"+command+":"), payload...))
	op.ID = ID("coreop_")
	_, err = s.DB.ExecContext(ctx, `INSERT INTO core_operations (operation_id,actor_id,client_request_id,node_id,command_type,fingerprint,payload,state,expires_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,'QUEUED',DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 60 SECOND),UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, op.ID, actor, client, node, command, hash, payload)
	if duplicate(err) {
		var old string
		err = s.DB.QueryRowContext(ctx, "SELECT operation_id,fingerprint FROM core_operations WHERE actor_id=? AND client_request_id=?", actor, client).Scan(&op.ID, &old)
		if err != nil {
			return
		}
		if old != hash {
			return op, true, ErrConflict
		}
		replay = true
	} else if err != nil {
		return
	}
	op, err = s.CoreOperation(ctx, op.ID)
	return
}
func (s *Store) CoreOperation(ctx context.Context, id string) (op CoreOperation, err error) {
	var result sql.NullString
	err = s.DB.QueryRowContext(ctx, "SELECT operation_id,actor_id,node_id,command_type,payload,state,result,expires_at,created_at FROM core_operations WHERE operation_id=?", id).Scan(&op.ID, &op.ActorID, &op.NodeID, &op.Command, &op.Payload, &op.State, &result, &op.ExpiresAt, &op.CreatedAt)
	if result.Valid {
		op.Result = json.RawMessage(result.String)
	}
	return
}
func (s *Store) MarkCoreSent(ctx context.Context, id string) (bool, error) {
	r, err := s.DB.ExecContext(ctx, "UPDATE core_operations SET state='SENT',updated_at=UTC_TIMESTAMP(6) WHERE operation_id=? AND state='QUEUED' AND expires_at>UTC_TIMESTAMP(6)", id)
	if err != nil {
		return false, err
	}
	n, err := r.RowsAffected()
	return n == 1, err
}
func (s *Store) CoreReply(ctx context.Context, node, id, status string, result []byte) error {
	if status != "COMPLETED" && status != "FAILED" && status != "UNKNOWN" && status != "PROCESSING" {
		return ErrConflict
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var current, assigned string
	var old sql.NullString
	if err = tx.QueryRowContext(ctx, "SELECT state,node_id,result FROM core_operations WHERE operation_id=? FOR UPDATE", id).Scan(&current, &assigned, &old); err != nil {
		return err
	}
	if assigned != node {
		return ErrUnauthorized
	}
	if current == "COMPLETED" || current == "FAILED" {
		var previous, received any
		if old.Valid && json.Unmarshal([]byte(old.String), &previous) == nil && json.Unmarshal(result, &received) == nil && reflect.DeepEqual(previous, received) {
			return nil
		}
		return ErrConflict
	}
	if current == "QUEUED" {
		return ErrConflict
	}
	if _, err = tx.ExecContext(ctx, "UPDATE core_operations SET state=?,result=?,updated_at=UTC_TIMESTAMP(6) WHERE operation_id=?", status, result, id); err != nil {
		return err
	}
	if status == "COMPLETED" {
		if err = walletTransferNoticeV206(ctx, tx, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *Store) MarkCoreUnknown(ctx context.Context, id string) error {
	_, err := s.DB.ExecContext(ctx, "UPDATE core_operations SET state='UNKNOWN',updated_at=UTC_TIMESTAMP(6) WHERE operation_id=? AND state IN ('SENT','PROCESSING')", id)
	return err
}
func (s *Store) DisconnectCore(ctx context.Context, node string) error {
	_, err := s.DB.ExecContext(ctx, "UPDATE core_operations SET state='UNKNOWN',updated_at=UTC_TIMESTAMP(6) WHERE node_id=? AND state IN ('SENT','PROCESSING')", node)
	return err
}
func (s *Store) RecoverCore(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, "UPDATE core_operations SET state='UNKNOWN',updated_at=UTC_TIMESTAMP(6) WHERE state IN ('SENT','PROCESSING')")
	return err
}
func (s *Store) RetryCore(ctx context.Context, id string) error {
	op, err := s.CoreOperation(ctx, id)
	if err != nil {
		return err
	}
	money := op.Command == "wallet.transfer" || strings.HasPrefix(op.Command, "wallet.escrow.")
	if op.Command != "player.resolve" && op.Command != "wallet.balance" && op.Command != "mailbox.query" && op.Command != "mailbox.create" && op.Command != "mailbox.revoke" && !money {
		return ErrConflict
	}
	if op.State == "SENT" || op.State == "PROCESSING" {
		return ErrConflict
	}
	if money && op.State != "UNKNOWN" && op.State != "QUEUED" {
		return ErrConflict
	}
	_, err = s.DB.ExecContext(ctx, "UPDATE core_operations SET state='QUEUED',result=NULL,expires_at=DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 60 SECOND),updated_at=UTC_TIMESTAMP(6) WHERE operation_id=? AND state=?", id, op.State)
	return err
}

type CorePlayerIdentity struct {
	PlayerUUID string `json:"playerUuid"`
	GameID     string `json:"gameId"`
	ServerID   string `json:"serverId"`
	Online     bool   `json:"online"`
	LastSeen   int64  `json:"lastSeen"`
}

func (s *Store) RememberCorePlayer(ctx context.Context, p CorePlayerIdentity) (string, error) {
	tx, err := s.beginAccountTx(ctx, "")
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	deleted, err := deletedUUIDTx(ctx, tx, p.PlayerUUID)
	if err != nil {
		return "", err
	}
	if deleted {
		return "", ErrSocialNotFound
	}
	ref := "player_" + Digest([]byte("deuterium-player:" + p.PlayerUUID))[:40]
	var registered string
	err = tx.QueryRowContext(ctx, "SELECT player_ref FROM identities WHERE server_uuid=? AND status<>'deleted'", p.PlayerUUID).Scan(&registered)
	if err == nil {
		ref = registered
	} else if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	var lastSeen any
	if p.LastSeen > 0 {
		lastSeen = time.UnixMilli(p.LastSeen).UTC()
	}
	if p.Online {
		lastSeen = time.Now().UTC()
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO core_player_directory VALUES (?,?,?,?,?) ON DUPLICATE KEY UPDATE player_ref=VALUES(player_ref),game_id=VALUES(game_id),node_id=VALUES(node_id),last_seen=CASE WHEN VALUES(last_seen) IS NULL THEN last_seen WHEN last_seen IS NULL THEN VALUES(last_seen) ELSE GREATEST(last_seen,VALUES(last_seen)) END`, ref, p.PlayerUUID, p.GameID, p.ServerID, lastSeen)
	if err == nil {
		err = tx.Commit()
	}
	return ref, err
}
func (s *Store) CoreRecipient(ctx context.Context, ref string) (uuid string, err error) {
	err = s.DB.QueryRowContext(ctx, "SELECT server_uuid FROM identities WHERE player_ref=? AND status='active'", ref).Scan(&uuid)
	if errors.Is(err, sql.ErrNoRows) {
		err = s.DB.QueryRowContext(ctx, "SELECT player_uuid FROM core_player_directory WHERE player_ref=?", ref).Scan(&uuid)
	}
	return
}

package store

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"time"
)

var ErrVerification = errors.New("invalid or expired verification")

func (s *Store) ReserveVerificationIssue(ctx context.Context, ip string) error {
	return s.reserveBudgets(ctx, map[string]int{Digest([]byte("verification-ip:" + ip)): 10})
}

type GameVerification struct {
	ID, TokenHash, CodeHash, Purpose, PlayerUUID, GameID, QQ, UserID, NodeID, State string
	ExpiresAt                                                                       time.Time
}

func (s *Store) NewGameVerification(ctx context.Context, v GameVerification) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	key := Digest([]byte(v.Purpose + ":" + v.PlayerUUID))
	if _, err = tx.ExecContext(ctx, "INSERT INTO game_verification_cooldowns VALUES (?,TIMESTAMP('2000-01-01')) ON DUPLICATE KEY UPDATE cooldown_key=VALUES(cooldown_key)", key); err != nil {
		return err
	}
	var next time.Time
	if err = tx.QueryRowContext(ctx, "SELECT expires_at FROM game_verification_cooldowns WHERE cooldown_key=? FOR UPDATE", key).Scan(&next); err != nil {
		return err
	}
	if next.After(time.Now().UTC()) {
		return ErrRateLimited
	}
	if _, err = tx.ExecContext(ctx, "UPDATE game_verification_cooldowns SET expires_at=DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 60 SECOND) WHERE cooldown_key=?", key); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE game_verifications SET state='EXPIRED' WHERE player_uuid=? AND purpose=? AND state IN ('PENDING','ACTIVE')", v.PlayerUUID, v.Purpose); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO game_verifications (verification_id,token_hash,code_hash,purpose,player_uuid,game_id,qq,user_id,node_id,state,expires_at,created_at) VALUES (?,?,?,?,?,?,?,?,?,'PENDING',?,UTC_TIMESTAMP(6))`, v.ID, v.TokenHash, v.CodeHash, v.Purpose, v.PlayerUUID, v.GameID, v.QQ, nullString(v.UserID), v.NodeID, v.ExpiresAt.UTC())
	if err != nil {
		return err
	}
	return tx.Commit()
}
func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
func (s *Store) ActivateGameVerification(ctx context.Context, id string) error {
	_, err := s.DB.ExecContext(ctx, "UPDATE game_verifications SET state='ACTIVE' WHERE verification_id=? AND state='PENDING' AND expires_at>UTC_TIMESTAMP(6)", id)
	return err
}
func (s *Store) ForgetVerificationCommand(ctx context.Context, id string) error {
	_, err := s.DB.ExecContext(ctx, "UPDATE core_operations SET payload='{}' WHERE operation_id=? AND command_type='verification.deliver'", id)
	return err
}
func (s *Store) CheckGameVerification(ctx context.Context, token, code, purpose string) (v GameVerification, err error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	var user sql.NullString
	var attempts int
	err = tx.QueryRowContext(ctx, `SELECT verification_id,token_hash,code_hash,purpose,player_uuid,game_id,qq,user_id,node_id,state,expires_at,attempts FROM game_verifications WHERE token_hash=? AND purpose=? FOR UPDATE`, token, purpose).Scan(&v.ID, &v.TokenHash, &v.CodeHash, &v.Purpose, &v.PlayerUUID, &v.GameID, &v.QQ, &user, &v.NodeID, &v.State, &v.ExpiresAt, &attempts)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrVerification
	}
	if err != nil {
		return
	}
	v.UserID = user.String
	if v.State != "ACTIVE" || !v.ExpiresAt.After(time.Now().UTC()) || attempts >= 5 {
		return v, ErrVerification
	}
	if subtle.ConstantTimeCompare([]byte(v.CodeHash), []byte(code)) != 1 {
		if _, err = tx.ExecContext(ctx, "UPDATE game_verifications SET attempts=attempts+1 WHERE verification_id=?", v.ID); err != nil {
			return
		}
		if err = tx.Commit(); err != nil {
			return
		}
		return v, ErrVerification
	}
	err = tx.Commit()
	return
}
func (s *Store) RegisterGameUser(ctx context.Context, v GameVerification, passwordHash string) (u User, err error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	if err = consumeVerification(ctx, tx, v); err != nil {
		return
	}
	u = User{ID: ID("user_"), PlayerRef: "player_" + Digest([]byte("deuterium-player:" + v.PlayerUUID))[:40], ServerUUID: v.PlayerUUID, GameID: v.GameID, QQ: v.QQ, PasswordHash: passwordHash, Status: "active", IdentityStatus: "bound"}
	_, err = tx.ExecContext(ctx, `INSERT INTO identities (id,player_ref,server_uuid,game_id,qq,password_hash,status,created_at,updated_at,legacy_fingerprint) VALUES (?,?,?,?,?,?,'active',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),'native')`, u.ID, u.PlayerRef, u.ServerUUID, u.GameID, u.QQ, u.PasswordHash)
	if duplicate(err) {
		return u, ErrConflict
	}
	if err != nil {
		return
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO identity_aliases VALUES (LOWER(?),?)", u.GameID, u.ID)
	if duplicate(err) {
		return u, ErrConflict
	}
	if err != nil {
		return
	}
	if u.QQ != u.GameID {
		_, err = tx.ExecContext(ctx, "INSERT INTO identity_aliases VALUES (?,?)", u.QQ, u.ID)
		if duplicate(err) {
			return u, ErrConflict
		}
		if err != nil {
			return
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE game_verifications SET user_id=? WHERE verification_id=?", u.ID, v.ID); err != nil {
		return
	}
	err = tx.Commit()
	return
}
func consumeVerification(ctx context.Context, tx *sql.Tx, v GameVerification) error {
	var state, hash string
	var expiry time.Time
	var attempts int
	if err := tx.QueryRowContext(ctx, "SELECT state,code_hash,expires_at,attempts FROM game_verifications WHERE verification_id=? FOR UPDATE", v.ID).Scan(&state, &hash, &expiry, &attempts); err != nil {
		return err
	}
	if state != "ACTIVE" || hash != v.CodeHash || !expiry.After(time.Now().UTC()) || attempts >= 5 {
		return ErrVerification
	}
	_, err := tx.ExecContext(ctx, "UPDATE game_verifications SET state='CONSUMED',consumed_at=UTC_TIMESTAMP(6) WHERE verification_id=?", v.ID)
	return err
}
func (s *Store) ResetGamePassword(ctx context.Context, v GameVerification, passwordHash string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = consumeVerification(ctx, tx, v); err != nil {
		return err
	}
	var status, uuid string
	if err = tx.QueryRowContext(ctx, "SELECT status,server_uuid FROM identities WHERE id=? FOR UPDATE", v.UserID).Scan(&status, &uuid); err != nil {
		return err
	}
	if status != "active" || uuid != v.PlayerUUID {
		return ErrUnauthorized
	}
	if _, err = tx.ExecContext(ctx, "UPDATE identities SET password_hash=?,updated_at=UTC_TIMESTAMP(6) WHERE id=?", passwordHash, v.UserID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE identity_sessions SET revoked_at=UTC_TIMESTAMP(6) WHERE user_id=? AND revoked_at IS NULL", v.UserID); err != nil {
		return err
	}
	return tx.Commit()
}

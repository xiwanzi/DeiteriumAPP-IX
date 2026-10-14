package store

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/notify"
	"strings"
	"time"
)

var ErrGameVerificationRequired = errors.New("game verification required")

type EmailVerification struct {
	ID, TokenHash, CodeHash, Email, Purpose, UserID, GameID string
	ExpiresAt                                               time.Time
	ProfileChecked                                          bool
	ProfileUUID                                             string
	ResendAt                                                time.Time
	IssueIP                                                 string
}

func (s *Store) NewEmailVerification(ctx context.Context, v EmailVerification, code string, key []byte) error {
	tx, err := s.beginAccountTx(ctx, "")
	if err != nil {
		return err
	}
	defer tx.Rollback()
	cooldown := Digest([]byte("email:" + v.Email))
	if _, err = tx.ExecContext(ctx, "INSERT IGNORE INTO game_verification_cooldowns VALUES(?,TIMESTAMP('2000-01-01'))", cooldown); err != nil {
		return err
	}
	var next time.Time
	if err = tx.QueryRowContext(ctx, "SELECT expires_at FROM game_verification_cooldowns WHERE cooldown_key=? FOR UPDATE", cooldown).Scan(&next); err != nil {
		return err
	}
	if err = checkEmailAddress(ctx, tx, v.Email); err != nil {
		return err
	}
	if next.After(time.Now().UTC()) {
		return rateLimitedUntil(next).withScope("RECIPIENT")
	}
	var count int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM email_verifications WHERE email=? AND created_at>DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 DAY)", v.Email).Scan(&count); err != nil {
		return err
	}
	if count >= 20 {
		var earliest time.Time
		if err = tx.QueryRowContext(ctx, "SELECT MIN(created_at) FROM email_verifications WHERE email=? AND created_at>DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 DAY)", v.Email).Scan(&earliest); err != nil {
			return err
		}
		return rateLimitedUntil(earliest.Add(24 * time.Hour)).withScope("RECIPIENT")
	}
	if v.IssueIP != "" {
		if err = s.reserveBudgetsTx(ctx, tx, map[string]int{Digest([]byte("verification-ip:" + v.IssueIP)): 10}); err != nil {
			var limited *RateLimitError
			if errors.As(err, &limited) {
				limited.Scope = "SOURCE"
			}
			return err
		}
	}
	resendAt := v.ResendAt
	if resendAt.IsZero() {
		resendAt = time.Now().UTC().Add(60 * time.Second)
	}
	if _, err = tx.ExecContext(ctx, "UPDATE game_verification_cooldowns SET expires_at=? WHERE cooldown_key=?", resendAt, cooldown); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE email_verifications SET state='EXPIRED' WHERE email=? AND purpose=? AND state='ACTIVE'", v.Email, v.Purpose); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO email_verifications(verification_id,token_hash,code_hash,email,purpose,user_id,game_id,state,expires_at,created_at,profile_checked,profile_uuid) VALUES(?,?,?,?,?,?,?,'ACTIVE',?,UTC_TIMESTAMP(6),?,?)`, v.ID, v.TokenHash, v.CodeHash, v.Email, v.Purpose, nullString(v.UserID), v.GameID, v.ExpiresAt, v.ProfileChecked, nullString(v.ProfileUUID)); err != nil {
		return err
	}
	payload := notify.VerificationMail{TemplateVersion: notify.VerificationTemplateVersion, Purpose: v.Purpose, GameID: v.GameID, Code: code}
	if payload.GameID == "" && v.UserID != "" {
		if err = tx.QueryRowContext(ctx, "SELECT game_id FROM identities WHERE id=?", v.UserID).Scan(&payload.GameID); err != nil {
			return err
		}
	}
	message, err := notify.RenderVerification(payload)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = enqueueMailTx(ctx, tx, "DM", "CODE_"+strings.ToUpper(v.Purpose), v.Email, message.Subject, string(encoded), v.ID, "", key)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) CheckEmailVerification(ctx context.Context, token, code, purpose string) (v EmailVerification, err error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return v, err
	}
	defer tx.Rollback()
	var state string
	var attempts int
	err = tx.QueryRowContext(ctx, `SELECT verification_id,token_hash,code_hash,email,purpose,COALESCE(user_id,''),game_id,state,attempts,expires_at,profile_checked,COALESCE(profile_uuid,'') FROM email_verifications WHERE token_hash=? AND purpose=? FOR UPDATE`, token, purpose).Scan(&v.ID, &v.TokenHash, &v.CodeHash, &v.Email, &v.Purpose, &v.UserID, &v.GameID, &state, &attempts, &v.ExpiresAt, &v.ProfileChecked, &v.ProfileUUID)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrVerification
	}
	if err != nil {
		return v, err
	}
	if state != "ACTIVE" || attempts >= 5 || !v.ExpiresAt.After(time.Now().UTC()) {
		return v, ErrVerification
	}
	if subtle.ConstantTimeCompare([]byte(v.CodeHash), []byte(code)) != 1 {
		if _, err = tx.ExecContext(ctx, "UPDATE email_verifications SET attempts=attempts+1 WHERE verification_id=?", v.ID); err != nil {
			return v, err
		}
		if err = tx.Commit(); err != nil {
			return v, err
		}
		return v, ErrVerification
	}
	return v, tx.Commit()
}

func consumeEmailVerification(ctx context.Context, tx *sql.Tx, v EmailVerification) error {
	result, err := tx.ExecContext(ctx, "UPDATE email_verifications SET state='CONSUMED' WHERE verification_id=? AND code_hash=? AND state='ACTIVE' AND attempts<5 AND expires_at>UTC_TIMESTAMP(6)", v.ID, v.CodeHash)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrVerification
	}
	return nil
}

func (s *Store) RegisterEmailUser(ctx context.Context, v EmailVerification, passwordHash string) (u User, err error) {
	if v.ProfileUUID != "" && !ValidAdmissionUUID(v.ProfileUUID) || v.ProfileChecked && v.ProfileUUID == "" {
		return u, ErrVerification
	}
	tx, err := s.beginAccountTx(ctx, "")
	if err != nil {
		return u, err
	}
	defer tx.Rollback()
	if v.ProfileUUID != "" {
		var occupied int
		if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM identities WHERE status<>'deleted' AND (profile_uuid=? OR server_uuid=?)", v.ProfileUUID, v.ProfileUUID).Scan(&occupied); err != nil {
			return u, err
		}
		if occupied > 0 {
			return u, ErrConflict
		}
	}
	if err = consumeEmailVerification(ctx, tx, v); err != nil {
		return u, err
	}
	u = User{ID: ID("user_"), PlayerRef: ID("player_"), Email: v.Email, GameID: v.GameID, ProfileUUID: v.ProfileUUID, DisplayName: v.GameID, PasswordHash: passwordHash, Status: "active", IdentityStatus: "unverified", GameVerified: false}
	_, err = tx.ExecContext(ctx, `INSERT INTO identities(id,player_ref,server_uuid,game_id,qq,email,game_verified,password_hash,status,created_at,updated_at,legacy_fingerprint,profile_uuid) VALUES(?,?,NULL,?,'',?,FALSE,?,'active',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),'native',?)`, u.ID, u.PlayerRef, u.GameID, u.Email, passwordHash, nullString(v.ProfileUUID))
	if duplicate(err) {
		return u, ErrConflict
	}
	if err != nil {
		return u, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO identity_aliases(alias_key,user_id) VALUES(LOWER(?),?)", u.GameID, u.ID)
	if duplicate(err) {
		return u, ErrConflict
	}
	if err != nil {
		return u, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO identity_aliases(alias_key,user_id) VALUES(?,?)", u.Email, u.ID)
	if duplicate(err) {
		return u, ErrConflict
	}
	if err != nil {
		return u, err
	}
	err = tx.Commit()
	return
}

func (s *Store) BindGameIdentity(ctx context.Context, userID string, v GameVerification) error {
	if v.UserID != userID || v.Purpose != "game_verify" {
		return ErrVerification
	}
	tx, err := s.beginAccountTx(ctx, userID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var verified bool
	var ref, currentGameID, profileUUID string
	if err = tx.QueryRowContext(ctx, "SELECT game_verified,player_ref,game_id,COALESCE(profile_uuid,'') FROM identities WHERE id=? FOR UPDATE", userID).Scan(&verified, &ref, &currentGameID, &profileUUID); err != nil {
		return err
	}
	if verified {
		return ErrConflict
	}
	if !strings.EqualFold(currentGameID, v.GameID) || profileUUID != "" && profileUUID != v.PlayerUUID {
		return ErrVerification
	}
	if err = consumeVerification(ctx, tx, v); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "UPDATE identities SET profile_uuid=?,server_uuid=?,game_id=?,qq=?,game_verified=TRUE,updated_at=UTC_TIMESTAMP(6) WHERE id=?", v.PlayerUUID, v.PlayerUUID, v.GameID, v.QQ, userID)
	if duplicate(err) {
		return ErrConflict
	}
	if err != nil {
		return err
	}
	for _, alias := range []string{strings.ToLower(v.GameID), v.QQ} {
		if alias == "" {
			continue
		}
		var owner string
		err = tx.QueryRowContext(ctx, "SELECT user_id FROM identity_aliases WHERE alias_key=?", alias).Scan(&owner)
		if err == nil && owner == userID {
			continue
		}
		if err == nil {
			return ErrConflict
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO identity_aliases(alias_key,user_id) VALUES(?,?)", alias, userID); err != nil {
			if duplicate(err) {
				return ErrConflict
			}
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE core_player_directory SET player_ref=? WHERE player_uuid=?", ref, v.PlayerUUID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,'account.game-verified',?,UTC_TIMESTAMP(6))", userID, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ResetEmailPassword(ctx context.Context, v EmailVerification, hash string) error {
	tx, err := s.beginAccountTx(ctx, v.UserID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = consumeEmailVerification(ctx, tx, v); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, "UPDATE identities SET password_hash=?,updated_at=UTC_TIMESTAMP(6) WHERE id=? AND email=? AND status='active'", hash, v.UserID, v.Email)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrUnauthorized
	}
	if _, err = tx.ExecContext(ctx, "UPDATE identity_sessions SET revoked_at=UTC_TIMESTAMP(6) WHERE user_id=? AND revoked_at IS NULL", v.UserID); err != nil {
		return err
	}
	return tx.Commit()
}

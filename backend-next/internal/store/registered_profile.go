package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// Registered profiles locate platform credentials only. They never provide a
// ServerUUID to callers that require proof of game ownership.
func (s *Store) UserByProfileUUID(ctx context.Context, uuid string) (u User, err error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,player_ref,game_id,qq,COALESCE(server_uuid,''),password_hash,status,COALESCE(email,''),game_verified,COALESCE(display_name,game_id),COALESCE(profile_uuid,'')
 FROM identities WHERE status<>'deleted' AND (profile_uuid=? OR (profile_uuid IS NULL AND server_uuid=? AND game_verified=TRUE))`, uuid, uuid)
	if err != nil {
		return u, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err = rows.Err(); err == nil {
			err = sql.ErrNoRows
		}
		return u, err
	}
	err = rows.Scan(&u.ID, &u.PlayerRef, &u.GameID, &u.QQ, &u.ServerUUID, &u.PasswordHash, &u.Status, &u.Email, &u.GameVerified, &u.DisplayName, &u.ProfileUUID)
	if err != nil {
		return u, err
	}
	if rows.Next() {
		return User{}, ErrNameConflict
	}
	if err = rows.Err(); err != nil {
		return User{}, err
	}
	u.IdentityStatus = "unverified"
	if u.GameVerified {
		u.IdentityStatus = "bound"
	}
	return u, nil
}

// The caller has checked the original password. Check it again by hash under
// the account lock so a concurrent reset, administrative edit or erasure cannot
// commit a rename based on credentials/profile association that are no longer current.
func (s *Store) SyncRegisteredName(ctx context.Context, user User, p AdmissionProfile, observedAt time.Time) error {
	if !ValidAdmissionUUID(p.UUID) || !ValidAdmissionName(p.Name) || user.ProfileUUID != p.UUID || strings.EqualFold(p.Name, "DIMA") || strings.EqualFold(p.Name, "DaoYu") {
		return ErrSocialInvalid
	}
	tx, err := s.beginAccountTx(ctx, user.ID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var old, uuid, hash, status, qq, email string
	var verified bool
	var previous sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT game_id,COALESCE(profile_uuid,''),password_hash,status,game_verified,qq,COALESCE(email,''),profile_observed_at FROM identities WHERE id=? FOR UPDATE`, user.ID).Scan(&old, &uuid, &hash, &status, &verified, &qq, &email, &previous)
	if err != nil {
		return err
	}
	if status != "active" || hash != user.PasswordHash || uuid != p.UUID || verified {
		return ErrCredentialChanged
	}
	if previous.Valid && previous.Time.After(observedAt) && old != p.Name {
		return ErrNameProofStale
	}
	if old != p.Name {
		var occupied int
		err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM identities WHERE id<>? AND status<>'deleted' AND LOWER(game_id)=LOWER(?)`, user.ID, p.Name).Scan(&occupied)
		if err != nil {
			return err
		}
		if occupied > 0 {
			return ErrNameConflict
		}
		var owner string
		err = tx.QueryRowContext(ctx, "SELECT user_id FROM identity_aliases WHERE alias_key=LOWER(?)", p.Name).Scan(&owner)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err == nil && owner != user.ID {
			return ErrNameConflict
		}
		if errors.Is(err, sql.ErrNoRows) {
			if _, err = tx.ExecContext(ctx, "INSERT INTO identity_aliases(alias_key,user_id) VALUES(LOWER(?),?)", p.Name, user.ID); err != nil {
				if duplicate(err) {
					return ErrNameConflict
				}
				return err
			}
		}
		if !strings.EqualFold(old, p.Name) && !strings.EqualFold(old, qq) && !strings.EqualFold(old, email) {
			if _, err = tx.ExecContext(ctx, "DELETE FROM identity_aliases WHERE user_id=? AND alias_key=LOWER(?)", user.ID, old); err != nil {
				return err
			}
		}
		if _, err = tx.ExecContext(ctx, "UPDATE identities SET game_id=?,admin_version=admin_version+1,updated_at=UTC_TIMESTAMP(6) WHERE id=?", p.Name, user.ID); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE game_verifications SET state='EXPIRED' WHERE user_id=? AND state IN ('PENDING','ACTIVE')", user.ID); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,'account.profile-name-sync',?,UTC_TIMESTAMP(6))", user.ID, user.ID); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE identities SET profile_observed_at=GREATEST(COALESCE(profile_observed_at,'1970-01-01'),?) WHERE id=?", observedAt.UTC(), user.ID); err != nil {
		return err
	}
	return tx.Commit()
}

package store

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"time"
)

type User struct {
	ID             string   `json:"userId"`
	PlayerRef      string   `json:"playerRef"`
	GameID         string   `json:"gameId"`
	QQ             string   `json:"qq"`
	IdentityStatus string   `json:"identityStatus"`
	Permissions    []string `json:"permissions,omitempty"`
	ServerUUID     string   `json:"-"`
	PasswordHash   string   `json:"-"`
	Status         string   `json:"-"`
}

type Session struct {
	User                  User
	TokenHash, Kind, CSRF string
	ExpiresAt             time.Time
}

func (s *Store) UserByAlias(ctx context.Context, alias string) (u User, err error) {
	err = s.DB.QueryRowContext(ctx, `SELECT i.id,i.player_ref,i.game_id,i.qq,i.server_uuid,i.password_hash,i.status
	 FROM identities i JOIN identity_aliases a ON a.user_id=i.id WHERE a.alias_key=?`, alias).Scan(&u.ID, &u.PlayerRef, &u.GameID, &u.QQ, &u.ServerUUID, &u.PasswordHash, &u.Status)
	u.IdentityStatus = "bound"
	return
}

func (s *Store) CreateSession(ctx context.Context, user User, hash, kind, csrf string, expiry time.Time, newPasswordHash string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var storedHash, status string
	if err = tx.QueryRowContext(ctx, "SELECT password_hash,status FROM identities WHERE id=? FOR UPDATE", user.ID).Scan(&storedHash, &status); err != nil {
		return err
	}
	if status != "active" {
		return ErrUnauthorized
	}
	if storedHash != user.PasswordHash {
		return ErrCredentialChanged
	}
	if newPasswordHash != "" {
		if _, err = tx.ExecContext(ctx, "UPDATE identities SET password_hash=?,updated_at=UTC_TIMESTAMP(6) WHERE id=?", newPasswordHash, user.ID); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO identity_sessions (token_hash,user_id,client_kind,csrf_token,created_at,expires_at) VALUES (?,?,?,?,UTC_TIMESTAMP(6),?)`, hash, user.ID, kind, csrf, expiry.UTC())
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Session(ctx context.Context, hash string) (v Session, err error) {
	err = s.DB.QueryRowContext(ctx, `SELECT i.id,i.player_ref,i.game_id,i.qq,i.server_uuid,i.status,s.client_kind,s.csrf_token,s.expires_at
	 FROM identity_sessions s JOIN identities i ON i.id=s.user_id WHERE s.token_hash=? AND s.revoked_at IS NULL AND s.expires_at>UTC_TIMESTAMP(6) AND i.status='active'`, hash).Scan(&v.User.ID, &v.User.PlayerRef, &v.User.GameID, &v.User.QQ, &v.User.ServerUUID, &v.User.Status, &v.Kind, &v.CSRF, &v.ExpiresAt)
	v.User.IdentityStatus = "bound"
	v.TokenHash = hash
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrUnauthorized
	}
	return
}

func (s *Store) Revoke(ctx context.Context, hash string) error {
	_, err := s.DB.ExecContext(ctx, "UPDATE identity_sessions SET revoked_at=UTC_TIMESTAMP(6) WHERE token_hash=? AND revoked_at IS NULL", hash)
	return err
}

func (s *Store) HasPermission(ctx context.Context, user, permission string) (bool, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM identity_permissions WHERE user_id=? AND (permission=? OR permission='platform.admin')", user, permission).Scan(&n)
	return n > 0, err
}

func (s *Store) Grant(ctx context.Context, user, permission string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "INSERT IGNORE INTO identity_permissions (user_id,permission) VALUES (?,?)", user, permission); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO audit_events_next (actor_id,action,resource_id,created_at) VALUES ('local-cli','permission.grant',?,UTC_TIMESTAMP(6))", user+":"+permission); err != nil {
		return err
	}
	return tx.Commit()
}

// Reserving attempts before hashing closes the parallel-request bypass. Persistent
// budgets survive restarts; the two keys are locked in a stable order.
func (s *Store) ReserveLogin(ctx context.Context, accountKey, ipKey string) error {
	return s.reserveBudgets(ctx, map[string]int{accountKey: 8, ipKey: 60})
}

func (s *Store) ReserveIdentityLogin(ctx context.Context, key string) error {
	return s.reserveBudgets(ctx, map[string]int{key: 8})
}

func (s *Store) reserveBudgets(ctx context.Context, limits map[string]int) error {
	keys := make([]string, 0, len(limits))
	for key := range limits {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, key := range keys {
		_, err = tx.ExecContext(ctx, `INSERT INTO auth_budgets VALUES (?,1,DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 15 MINUTE))
		 ON DUPLICATE KEY UPDATE attempts=IF(expires_at<=UTC_TIMESTAMP(6),1,attempts+1),expires_at=IF(expires_at<=UTC_TIMESTAMP(6),DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 15 MINUTE),expires_at)`, key)
		if err != nil {
			return err
		}
		var n int
		if err = tx.QueryRowContext(ctx, "SELECT attempts FROM auth_budgets WHERE budget_key=?", key).Scan(&n); err != nil {
			return err
		}
		if n > limits[key] {
			return ErrRateLimited
		}
	}
	return tx.Commit()
}

func (s *Store) ClearAccountBudget(ctx context.Context, key string) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM auth_budgets WHERE budget_key=?", key)
	return err
}

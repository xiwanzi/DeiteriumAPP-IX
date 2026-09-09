package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

type AdminAccountV206 struct {
	UserID    string    `json:"userId"`
	PlayerRef string    `json:"playerRef"`
	GameID    string    `json:"gameId"`
	QQ        string    `json:"qq"`
	Status    string    `json:"status"`
	Admin     bool      `json:"admin"`
	BanReason string    `json:"banReason"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
}

// All role/status changes use one small guard row before locking identities.
// This also serializes two administrators attempting to disable each other.
func requirePlatformAdminTxV206(ctx context.Context, tx *sql.Tx, actor string) error {
	var guard int
	if err := tx.QueryRowContext(ctx, "SELECT id FROM admin_account_guard WHERE id=1 FOR UPDATE").Scan(&guard); err != nil {
		return err
	}
	var status string
	if err := tx.QueryRowContext(ctx, "SELECT status FROM identities WHERE id=? FOR UPDATE", actor).Scan(&status); err != nil {
		return err
	}
	var n int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM identity_permissions WHERE user_id=? AND permission='platform.admin'", actor).Scan(&n); err != nil {
		return err
	}
	if status != "active" || n != 1 {
		return ErrUnauthorized
	}
	return nil
}

func (s *Store) AdminAccountsV206(ctx context.Context, search, status string, offset, limit int) ([]AdminAccountV206, int, error) {
	if offset < 0 || limit < 1 || limit > 100 || len(search) > 100 || !catalogEnum(status, "", "active", "disabled", "locked") {
		return nil, 0, ErrSocialInvalid
	}
	search = "%" + strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(search), "!", "!!"), "%", "!%"), "_", "!_") + "%"
	where := ` WHERE (?='' OR i.status=?) AND (i.game_id LIKE ? ESCAPE '!' OR i.qq LIKE ? ESCAPE '!' OR i.player_ref LIKE ? ESCAPE '!')`
	args := []any{status, status, search, search, search}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM identities i"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT i.id,i.player_ref,i.game_id,i.qq,i.status,EXISTS(SELECT 1 FROM identity_permissions p WHERE p.user_id=i.id AND p.permission='platform.admin'),i.ban_reason,i.admin_version,i.created_at FROM identities i`+where+" ORDER BY i.created_at DESC,i.id LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	result := []AdminAccountV206{}
	for rows.Next() {
		var a AdminAccountV206
		if err = rows.Scan(&a.UserID, &a.PlayerRef, &a.GameID, &a.QQ, &a.Status, &a.Admin, &a.BanReason, &a.Version, &a.CreatedAt); err != nil {
			return nil, 0, err
		}
		result = append(result, a)
	}
	return result, total, rows.Err()
}

type AdminAccountChangeV206 struct {
	ClientRequestID string `json:"clientRequestId"`
	ExpectedVersion int64  `json:"expectedVersion"`
	Action          string `json:"action"`
	Reason          string `json:"reason"`
}

func (s *Store) ChangeAdminAccountV206(ctx context.Context, actor, target string, input AdminAccountChangeV206) (json.RawMessage, error) {
	if !ValidSocialID(target) || input.ExpectedVersion < 1 || !catalogEnum(input.Action, "grant-admin", "revoke-admin", "ban", "unban") || !ValidSocialText(input.Reason, 0, 500) {
		return nil, ErrSocialInvalid
	}
	if actor == target {
		return nil, catalogError(409, "SELF_ACCOUNT_CHANGE", "请由另一位管理员调整你的权限或账号状态。")
	}
	if input.Action == "ban" && strings.TrimSpace(input.Reason) == "" {
		return nil, catalogError(400, "BAN_REASON_REQUIRED", "请填写封禁原因。")
	}
	return s.socialMutate(ctx, actor, "account.manage:"+target, input.ClientRequestID, input, func(tx *sql.Tx) (any, error) {
		if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
			return nil, err
		}
		var status string
		var version int64
		if err := tx.QueryRowContext(ctx, "SELECT status,admin_version FROM identities WHERE id=? FOR UPDATE", target).Scan(&status, &version); err != nil {
			return nil, socialMissing(err)
		}
		if version != input.ExpectedVersion {
			return nil, ErrSocialVersion
		}
		if input.Action == "ban" || input.Action == "revoke-admin" {
			var remaining int
			if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM identities i JOIN identity_permissions p ON p.user_id=i.id AND p.permission='platform.admin' WHERE i.status='active' AND i.id<>?", target).Scan(&remaining); err != nil {
				return nil, err
			}
			if remaining == 0 {
				return nil, catalogError(409, "LAST_ADMIN", "请至少保留一位可用的管理员。")
			}
		}
		var err error
		switch input.Action {
		case "grant-admin":
			if status != "active" {
				return nil, catalogError(409, "ACCOUNT_DISABLED", "请先解封账号，再授予管理权限。")
			}
			_, err = tx.ExecContext(ctx, "INSERT IGNORE INTO identity_permissions VALUES(?,'platform.admin')", target)
		case "revoke-admin":
			_, err = tx.ExecContext(ctx, "DELETE FROM identity_permissions WHERE user_id=? AND permission='platform.admin'", target)
		case "ban":
			_, err = tx.ExecContext(ctx, "UPDATE identities SET status='disabled',ban_reason=? WHERE id=?", strings.TrimSpace(input.Reason), target)
		case "unban":
			_, err = tx.ExecContext(ctx, "UPDATE identities SET status='active',ban_reason='' WHERE id=?", target)
		}
		if err != nil {
			return nil, err
		}
		if input.Action == "ban" || input.Action == "revoke-admin" {
			if _, err = tx.ExecContext(ctx, "UPDATE identity_sessions SET revoked_at=COALESCE(revoked_at,UTC_TIMESTAMP(6)) WHERE user_id=?", target); err != nil {
				return nil, err
			}
		}
		if _, err = tx.ExecContext(ctx, "UPDATE identities SET admin_version=admin_version+1,updated_at=UTC_TIMESTAMP(6) WHERE id=?", target); err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,?,?,UTC_TIMESTAMP(6))", actor, "account."+input.Action, target); err != nil {
			return nil, err
		}
		return map[string]any{"userId": target, "version": version + 1, "action": input.Action}, nil
	})
}

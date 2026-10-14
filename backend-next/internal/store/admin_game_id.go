package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

var adminGameIDPattern = regexp.MustCompile(`^[A-Za-z0-9_]{1,32}$`)

type AdminGameIDChange struct {
	ClientRequestID string `json:"clientRequestId"`
	ExpectedVersion int64  `json:"expectedVersion"`
	GameID          string `json:"gameId"`
}

type AdminGameIdentity struct {
	GameID, UUID, Status string
	Verified             bool
	Version              int64
}

func (s *Store) AdminGameIdentity(ctx context.Context, target string) (out AdminGameIdentity, err error) {
	err = s.DB.QueryRowContext(ctx, "SELECT game_id,COALESCE(server_uuid,''),status,game_verified,admin_version FROM identities WHERE id=?", target).Scan(&out.GameID, &out.UUID, &out.Status, &out.Verified, &out.Version)
	return out, socialMissing(err)
}

// resolvedUUID is supplied only by the server's Core resolver, never by the client.
func (s *Store) RenameAccountGameID(ctx context.Context, actor, target string, in AdminGameIDChange, resolvedUUID string) (json.RawMessage, error) {
	in.GameID = strings.TrimSpace(in.GameID)
	if !ValidSocialID(target) || in.ExpectedVersion < 1 || !adminGameIDPattern.MatchString(in.GameID) || strings.EqualFold(in.GameID, "DIMA") || strings.EqualFold(in.GameID, "DaoYu") {
		return nil, ErrSocialInvalid
	}
	if actor == target {
		return nil, catalogError(409, "SELF_ACCOUNT_CHANGE", "请由另一位超级管理员修改你的游戏 ID。")
	}
	return s.socialMutate(ctx, actor, "account.game-id:"+target, in.ClientRequestID, in, func(tx *sql.Tx) (any, error) {
		if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
			return nil, err
		}
		var old, uuid, status, qq string
		var verified bool
		var version int64
		if err := tx.QueryRowContext(ctx, "SELECT game_id,COALESCE(server_uuid,''),status,qq,game_verified,admin_version FROM identities WHERE id=? FOR UPDATE", target).Scan(&old, &uuid, &status, &qq, &verified, &version); err != nil {
			return nil, socialMissing(err)
		}
		if status == "deleted" {
			return nil, catalogError(409, "ACCOUNT_DELETED", "该账号已永久注销。")
		}
		if version != in.ExpectedVersion {
			return nil, ErrSocialVersion
		}
		if verified && (uuid == "" || uuid != resolvedUUID) {
			return nil, catalogError(409, "GAME_ID_IDENTITY_MISMATCH", "新游戏 ID 必须属于原来绑定的游戏身份，不能转移账号或资产。")
		}
		var occupied int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM identities WHERE LOWER(game_id)=LOWER(?) AND id<>? AND status<>'deleted'", in.GameID, target).Scan(&occupied); err != nil {
			return nil, err
		}
		var aliasOwner string
		err := tx.QueryRowContext(ctx, "SELECT user_id FROM identity_aliases WHERE alias_key=LOWER(?)", in.GameID).Scan(&aliasOwner)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if occupied > 0 || err == nil && aliasOwner != target {
			return nil, catalogError(409, "GAME_ID_IN_USE", "该游戏 ID 已被其他账号使用。")
		}
		if old == in.GameID {
			return map[string]any{"userId": target, "gameId": old, "previousGameId": old, "version": version, "changed": false}, nil
		}
		if errors.Is(err, sql.ErrNoRows) {
			if _, err = tx.ExecContext(ctx, "INSERT INTO identity_aliases(alias_key,user_id) VALUES(LOWER(?),?)", in.GameID, target); err != nil {
				if duplicate(err) {
					return nil, catalogError(409, "GAME_ID_IN_USE", "该游戏 ID 已被其他账号使用。")
				}
				return nil, err
			}
		}
		if !strings.EqualFold(old, in.GameID) && old != qq {
			if _, err = tx.ExecContext(ctx, "DELETE FROM identity_aliases WHERE user_id=? AND alias_key=LOWER(?)", target, old); err != nil {
				return nil, err
			}
		}
		if _, err = tx.ExecContext(ctx, "UPDATE identities SET profile_uuid=IF(game_verified,server_uuid,?),profile_observed_at=UTC_TIMESTAMP(6),game_id=?,admin_version=admin_version+1,updated_at=UTC_TIMESTAMP(6) WHERE id=?", nullString(resolvedUUID), in.GameID, target); err != nil {
			if duplicate(err) {
				return nil, catalogError(409, "GAME_ID_IN_USE", "该正版账号已被其他平台账号登记，请联系支持人员核对。")
			}
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE game_verifications SET state='EXPIRED' WHERE user_id=? AND state IN ('PENDING','ACTIVE')", target); err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE identity_sessions SET revoked_at=COALESCE(revoked_at,UTC_TIMESTAMP(6)) WHERE user_id=?", target); err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,'account.game-id',?,UTC_TIMESTAMP(6))", actor, target); err != nil {
			return nil, err
		}
		return map[string]any{"userId": target, "gameId": in.GameID, "previousGameId": old, "version": version + 1, "changed": true}, nil
	})
}

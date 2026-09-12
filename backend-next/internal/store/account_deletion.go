package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
)

type AccountDeletionRequest struct {
	ClientRequestID string `json:"clientRequestId"`
	ExpectedVersion int64  `json:"expectedVersion"`
}

type AccountDeletionProof struct{ ActorID, SessionHash, CredentialHash string }

func (s *Store) DeleteAccount(ctx context.Context, proof AccountDeletionProof, target string, input AccountDeletionRequest) (json.RawMessage, error) {
	if !ValidSocialID(target) || !ValidSocialID(input.ClientRequestID) || input.ExpectedVersion < 1 {
		return nil, ErrSocialInvalid
	}
	if target == proof.ActorID {
		return nil, catalogError(409, "SELF_ACCOUNT_CHANGE", "不能永久注销当前登录的管理员账号。")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var guard int
	if err = tx.QueryRowContext(ctx, "SELECT id FROM account_lifecycle_guard WHERE id=1 FOR UPDATE").Scan(&guard); err != nil {
		return nil, err
	}
	if err = requirePlatformAdminTxV206(ctx, tx, proof.ActorID); err != nil {
		return nil, err
	}
	var currentHash string
	if err = tx.QueryRowContext(ctx, "SELECT password_hash FROM identities WHERE id=?", proof.ActorID).Scan(&currentHash); err != nil {
		return nil, err
	}
	if currentHash != proof.CredentialHash || proof.CredentialHash == "" {
		return nil, catalogError(409, "ADMIN_CREDENTIAL_CHANGED", "管理员密码已改变，请重新输入当前密码确认。")
	}
	var sessionUser string
	err = tx.QueryRowContext(ctx, "SELECT user_id FROM identity_sessions WHERE token_hash=? AND revoked_at IS NULL AND expires_at>UTC_TIMESTAMP(6) FOR UPDATE", proof.SessionHash).Scan(&sessionUser)
	if errors.Is(err, sql.ErrNoRows) || err == nil && sessionUser != proof.ActorID {
		return nil, ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}

	// The password is absent from both the idempotency fingerprint and response.
	action := "account.delete:" + target
	fingerprint := Digest([]byte(target + ":" + catalogJSON(input)))
	if _, err = tx.ExecContext(ctx, `INSERT INTO social_requests_v2(actor_id,action,request_id,fingerprint,created_at) VALUES(?,?,?,?,UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE request_id=VALUES(request_id)`, proof.ActorID, action, input.ClientRequestID, fingerprint); err != nil {
		return nil, err
	}
	var previous string
	var response sql.NullString
	if err = tx.QueryRowContext(ctx, "SELECT fingerprint,response_json FROM social_requests_v2 WHERE actor_id=? AND action=? AND request_id=? FOR UPDATE", proof.ActorID, action, input.ClientRequestID).Scan(&previous, &response); err != nil {
		return nil, err
	}
	if previous != fingerprint {
		return nil, ErrConflict
	}
	if response.Valid {
		return json.RawMessage(response.String), nil
	}
	var u User
	var version int64
	err = tx.QueryRowContext(ctx, "SELECT id,player_ref,server_uuid,game_id,qq,status,admin_version FROM identities WHERE id=? FOR UPDATE", target).Scan(&u.ID, &u.PlayerRef, &u.ServerUUID, &u.GameID, &u.QQ, &u.Status, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSocialNotFound
	}
	if err != nil {
		return nil, err
	}
	if u.Status == "deleted" {
		return nil, catalogError(409, "ACCOUNT_DELETED", "该账号已永久注销，无法恢复。")
	}
	if version != input.ExpectedVersion {
		return nil, ErrSocialVersion
	}
	if err = accountDeletionAllowedTx(ctx, tx, u); err != nil {
		return nil, err
	}
	if err = eraseAccountPresentationTx(ctx, tx, u); err != nil {
		return nil, err
	}
	var tombstone [16]byte
	if _, err = rand.Read(tombstone[:]); err != nil {
		return nil, err
	}
	tombstone[6] = (tombstone[6] & 15) | 64
	tombstone[8] = (tombstone[8] & 63) | 128
	h := hex.EncodeToString(tombstone[:])
	uuid := h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
	if _, err = tx.ExecContext(ctx, "UPDATE identities SET status='deleted',game_id=?,qq='',password_hash='',server_uuid=?,legacy_fingerprint='deleted',ban_reason='',admin_version=admin_version+1,updated_at=UTC_TIMESTAMP(6) WHERE id=?", DeletedAccountName, uuid, target); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO account_deletions(user_id,player_ref,original_uuid,deleted_at) VALUES(?,?,?,UTC_TIMESTAMP(6))", target, u.PlayerRef, u.ServerUUID); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,'account.delete',?,UTC_TIMESTAMP(6))", proof.ActorID, target); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(map[string]any{"userId": target, "deleted": true, "version": version + 1})
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE social_requests_v2 SET response_json=? WHERE actor_id=? AND action=? AND request_id=?", encoded, proof.ActorID, action, input.ClientRequestID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return encoded, nil
}

func accountDeletionAllowedTx(ctx context.Context, tx *sql.Tx, u User) error {
	checks := []struct {
		query, code, message string
		args                 []any
	}{
		{`SELECT COUNT(*) FROM identities i JOIN identity_permissions p ON p.user_id=i.id AND p.permission='platform.admin' WHERE i.status='active' AND i.id<>?`, "LAST_ADMIN", "请至少保留一位可用的管理员。", []any{u.ID}},
		{`SELECT COUNT(*) FROM commerce_resources_v2 WHERE (owner_id=? OR payee_id=?) AND (pending_operation_id IS NOT NULL OR state NOT IN ('CONFIRMED','CLAIMED','REFUNDED','CANCELLED') OR funds_state NOT IN ('SETTLED','REFUNDED','UNPAID') OR (funds_state='UNPAID' AND state<>'CANCELLED'))`, "ACCOUNT_HAS_OPEN_TRANSACTIONS", "该账号有未完成的订单或委托，请处理完毕后再注销。", []any{u.ID, u.ID}},
		{`SELECT COUNT(*) FROM commerce_interventions_v2 c JOIN commerce_resources_v2 r ON r.resource_id=c.resource_id WHERE (r.owner_id=? OR r.payee_id=?) AND c.state NOT IN ('RESOLVED','WITHDRAWN')`, "ACCOUNT_HAS_OPEN_TRANSACTIONS", "该账号仍有平台介入待处理。", []any{u.ID, u.ID}},
		{`SELECT COUNT(*) FROM commerce_refunds_v2 f JOIN commerce_resources_v2 r ON r.resource_id=f.resource_id WHERE (r.owner_id=? OR r.payee_id=?) AND f.state IN ('REQUESTED','PROCESSING')`, "ACCOUNT_HAS_OPEN_TRANSACTIONS", "该账号仍有退款待处理。", []any{u.ID, u.ID}},
		{`SELECT COUNT(*) FROM core_operations WHERE command_type='wallet.transfer' AND state NOT IN ('COMPLETED','FAILED') AND (actor_id=? OR JSON_UNQUOTE(JSON_EXTRACT(payload,'$.fromUuid'))=? OR JSON_UNQUOTE(JSON_EXTRACT(payload,'$.toUuid'))=?)`, "ACCOUNT_HAS_PENDING_TRANSFER", "该账号存在处理中的转账或结果尚未明确的转账。", []any{u.ID, u.ServerUUID, u.ServerUUID}},
		{`SELECT COUNT(*) FROM catalog_records_v2 WHERE kind='store' AND owner_id=? AND state<>'DELETED'`, "ACCOUNT_OWNS_STORE", "该账号仍持有商店，请先完成商店归属处理。", []any{u.ID}},
	}
	for i, c := range checks {
		var count int
		if err := tx.QueryRowContext(ctx, c.query, c.args...).Scan(&count); err != nil {
			return err
		}
		if i == 0 && count == 0 || i > 0 && count > 0 {
			return catalogError(409, c.code, c.message)
		}
	}
	return nil
}

func eraseAccountPresentationTx(ctx context.Context, tx *sql.Tx, u User) error {
	// Resolve and invalidate snapshots of private messages before deleting their
	// original conversation, including forwards kept in unrelated conversations.
	privateIDs := `SELECT m.message_id FROM social_messages_v2 m JOIN social_conversations_v2 c ON c.conversation_id=m.conversation_id WHERE c.user_lo=? OR c.user_hi=?`
	for _, table := range []string{"social_messages_v2", "public_chat_metadata_v2"} {
		columns := []string{"reply_json"}
		if table == "social_messages_v2" {
			columns = append(columns, "forwarded_json")
		}
		for _, column := range columns {
			set := column + `=JSON_OBJECT('messageId',JSON_UNQUOTE(JSON_EXTRACT(` + column + `,'$.messageId')),'sender',JSON_OBJECT('gameId','','playerRef','','registered',FALSE),'content','','availability','UNAVAILABLE')`
			if column == "forwarded_json" {
				set = "content='原消息不可见'," + set
			}
			query := `UPDATE ` + table + ` SET ` + set + ` WHERE JSON_UNQUOTE(JSON_EXTRACT(` + column + `,'$.sender.playerRef'))=? OR JSON_UNQUOTE(JSON_EXTRACT(` + column + `,'$.messageId')) IN (` + privateIDs + `)`
			if _, err := tx.ExecContext(ctx, query, u.PlayerRef, u.ID, u.ID); err != nil {
				return err
			}
		}
	}
	queries := []struct {
		sql  string
		args []any
	}{
		{`DELETE r FROM social_requests_v2 r JOIN social_conversations_v2 c ON LOCATE(c.conversation_id,r.response_json)>0 WHERE c.user_lo=? OR c.user_hi=?`, []any{u.ID, u.ID}},
		{`DELETE n FROM social_notifications_v2 n JOIN social_conversations_v2 c ON JSON_UNQUOTE(JSON_EXTRACT(n.target_json,'$.referenceId'))=c.conversation_id WHERE c.user_lo=? OR c.user_hi=?`, []any{u.ID, u.ID}},
		{`DELETE n FROM social_notifications_v2 n JOIN chat_messages_next m ON JSON_UNQUOTE(JSON_EXTRACT(n.target_json,'$.referenceId'))=m.message_id WHERE m.sender_ref=? OR m.sender_uuid=?`, []any{u.PlayerRef, u.ServerUUID}},
		{`UPDATE social_notifications_v2 n JOIN commerce_resources_v2 r ON JSON_UNQUOTE(JSON_EXTRACT(n.target_json,'$.referenceId')) IN (r.resource_id,r.refund_id,r.intervention_case_id) SET n.body='该历史记录涉及已注销用户，请查看交易详情。' WHERE r.owner_id=? OR r.payee_id=?`, []any{u.ID, u.ID}},
		{`DELETE m FROM social_messages_v2 m JOIN social_conversations_v2 c ON c.conversation_id=m.conversation_id WHERE c.user_lo=? OR c.user_hi=?`, []any{u.ID, u.ID}},
		{`DELETE m FROM social_members_v2 m JOIN social_conversations_v2 c ON c.conversation_id=m.conversation_id WHERE c.user_lo=? OR c.user_hi=?`, []any{u.ID, u.ID}},
		{`DELETE FROM social_conversations_v2 WHERE user_lo=? OR user_hi=?`, []any{u.ID, u.ID}},
		{`DELETE FROM social_follows_v2 WHERE follower_id=? OR followed_id=?`, []any{u.ID, u.ID}},
		{`DELETE d FROM core_chat_deliveries d JOIN chat_messages_next m ON m.sequence_id=d.message_sequence WHERE m.sender_ref=? OR m.sender_uuid=?`, []any{u.PlayerRef, u.ServerUUID}},
		{`DELETE p FROM public_chat_metadata_v2 p JOIN chat_messages_next m ON m.message_id=p.message_id WHERE m.sender_ref=? OR m.sender_uuid=?`, []any{u.PlayerRef, u.ServerUUID}},
		{`DELETE FROM chat_messages_next WHERE sender_ref=? OR sender_uuid=?`, []any{u.PlayerRef, u.ServerUUID}},
		{`UPDATE public_chat_metadata_v2 SET mentioned_refs_json=JSON_REMOVE(mentioned_refs_json,JSON_UNQUOTE(JSON_SEARCH(mentioned_refs_json,'one',?))) WHERE JSON_SEARCH(mentioned_refs_json,'one',?) IS NOT NULL`, []any{u.PlayerRef, u.PlayerRef}},
		{`UPDATE chat_messages_next SET content=REGEXP_REPLACE(content,?,?) WHERE content REGEXP ?`, []any{"(?i)@" + regexp.QuoteMeta(u.GameID) + "(?![A-Za-z0-9_])", "@" + DeletedAccountName, "(?i)@" + regexp.QuoteMeta(u.GameID) + "(?![A-Za-z0-9_])"}},
		{`DELETE FROM social_requests_v2 WHERE actor_id=? OR LOCATE(?,response_json)>0`, []any{u.ID, u.PlayerRef}},
		{`DELETE b FROM asset_bindings_v2 b JOIN catalog_records_v2 c ON c.resource_id=b.business_ref WHERE c.kind='listing' AND c.owner_id=? AND b.business_type='MARKET_LISTING'`, []any{u.ID}},
		{`DELETE FROM asset_bindings_v2 WHERE business_type='PROFILE' AND business_ref=?`, []any{u.ID}},
		{`DELETE q FROM catalog_quotes_v2 q JOIN catalog_records_v2 c ON LOCATE(c.resource_id,q.snapshot)>0 WHERE c.kind='listing' AND c.owner_id=?`, []any{u.ID}},
		{`UPDATE catalog_records_v2 SET state='DELETED',body='{}',published_body=NULL,published_version=NULL,published_at=NULL,available_stock=0,title='',category_ref='',brand_ref='',version=version+1,updated_at=UTC_TIMESTAMP(6) WHERE kind='listing' AND owner_id=?`, []any{u.ID}},
		{`UPDATE catalog_members_v2 SET active=FALSE,version=version+1 WHERE user_id=?`, []any{u.ID}},
		{`UPDATE asset_uploads_v2 a SET alt_text='',expires_at=LEAST(expires_at,UTC_TIMESTAMP(6)),lifecycle_checked_at=NULL,removed_at=IF(NOT EXISTS(SELECT 1 FROM asset_bindings_v2 b WHERE b.asset_id=a.asset_id),UTC_TIMESTAMP(6),removed_at) WHERE user_id=?`, []any{u.ID}},
		{`DELETE FROM game_verifications WHERE user_id=? OR player_uuid=?`, []any{u.ID, u.ServerUUID}},
		{`DELETE FROM core_player_directory WHERE player_uuid=?`, []any{u.ServerUUID}},
	}
	for _, q := range queries {
		if _, err := tx.ExecContext(ctx, q.sql, q.args...); err != nil {
			return err
		}
	}
	for _, table := range []string{"identity_aliases", "identity_permissions", "identity_sessions", "social_profiles_v2", "social_notifications_v2", "social_notification_preferences_v2", "social_send_budgets_v2", "catalog_cart_items_v2", "catalog_carts_v2", "catalog_quotes_v2", "catalog_mutations_v2", "ai_messages_v2", "ai_profiles_v2", "ai_requests_v2", "ai_conversations_v2", "ai_quota_v2", "ai_entitlements_v206", "personal_record_visibility_v203"} {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE user_id=?", u.ID); err != nil {
			return err
		}
	}
	return nil
}

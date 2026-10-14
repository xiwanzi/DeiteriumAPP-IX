package store

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type ProfileReleaseRequest struct {
	ClientRequestID    string `json:"clientRequestId"`
	ExpectedVersion    int64  `json:"expectedVersion"`
	ClaimantEmail      string `json:"claimantEmail"`
	EvidenceType       string `json:"evidenceType"`
	EvidenceReference  string `json:"evidenceReference"`
	Reason             string `json:"reason"`
	OwnershipConfirmed bool   `json:"ownershipConfirmed"`
}

type ProfileReleaseProof struct {
	AccountDeletionProof
	GameTokenHash, GameCodeHash string
}

func requireAdminSessionProofTx(ctx context.Context, tx *sql.Tx, proof AccountDeletionProof) error {
	if err := requirePlatformAdminTxV206(ctx, tx, proof.ActorID); err != nil {
		return err
	}
	var hash string
	if err := tx.QueryRowContext(ctx, "SELECT password_hash FROM identities WHERE id=?", proof.ActorID).Scan(&hash); err != nil {
		return err
	}
	if proof.CredentialHash == "" || hash != proof.CredentialHash {
		return catalogError(409, "ADMIN_CREDENTIAL_CHANGED", "管理员密码已改变，请重新登录后确认。")
	}
	var user string
	err := tx.QueryRowContext(ctx, `SELECT user_id FROM identity_sessions WHERE token_hash=? AND revoked_at IS NULL AND client_kind IN ('app','web') AND (expires_at>UTC_TIMESTAMP(6) OR (expires_at IS NULL AND client_kind='app')) FOR UPDATE`, proof.SessionHash).Scan(&user)
	if errors.Is(err, sql.ErrNoRows) || err == nil && user != proof.ActorID {
		return ErrUnauthorized
	}
	return err
}

// A rare admin resolution takes the existing lifecycle gate exclusively. All
// registration/binding writes take it shared, so nobody can pass a reservation
// check before release and insert an identity after release. No network/KDF work
// runs in this transaction; game admission reads do not take this gate.
func (s *Store) ReleaseRegisteredProfile(ctx context.Context, target string, in ProfileReleaseRequest, proof ProfileReleaseProof) (json.RawMessage, error) {
	var err error
	if in.ClaimantEmail, err = NormalizeEmail(in.ClaimantEmail); err != nil {
		return nil, err
	}
	in.EvidenceReference, in.Reason = strings.TrimSpace(in.EvidenceReference), strings.TrimSpace(in.Reason)
	if !ValidSocialID(target) || !ValidSocialID(in.ClientRequestID) || in.ExpectedVersion < 1 || !in.OwnershipConfirmed || !ValidSocialText(in.EvidenceReference, 3, 240) || !ValidSocialText(in.Reason, 3, 500) || (in.EvidenceType != "GAME_CODE" && in.EvidenceType != "SUPPORT_REVIEW") {
		return nil, ErrSocialInvalid
	}
	if target == proof.ActorID {
		return nil, catalogError(409, "SELF_ACCOUNT_CHANGE", "请由另一位超级管理员处理身份占用。")
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var guard int
	if err = tx.QueryRowContext(ctx, "SELECT id FROM account_lifecycle_guard WHERE id=1 FOR UPDATE").Scan(&guard); err != nil {
		return nil, err
	}
	if err = requireAdminSessionProofTx(ctx, tx, proof.AccountDeletionProof); err != nil {
		return nil, err
	}
	action, fingerprint := "account.profile-release:"+target, Digest([]byte(catalogJSON(in)))
	if _, err = tx.ExecContext(ctx, `INSERT INTO social_requests_v2(actor_id,action,request_id,fingerprint,created_at) VALUES(?,?,?,?,UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE request_id=VALUES(request_id)`, proof.ActorID, action, in.ClientRequestID, fingerprint); err != nil {
		return nil, err
	}
	var old string
	var response sql.NullString
	if err = tx.QueryRowContext(ctx, "SELECT fingerprint,response_json FROM social_requests_v2 WHERE actor_id=? AND action=? AND request_id=? FOR UPDATE", proof.ActorID, action, in.ClientRequestID).Scan(&old, &response); err != nil {
		return nil, err
	}
	if old != fingerprint {
		return nil, ErrConflict
	}
	if response.Valid {
		return json.RawMessage(response.String), nil
	}
	var gameID, uuid, serverUUID, email, qq, status string
	var verified bool
	var version int64
	err = tx.QueryRowContext(ctx, `SELECT game_id,COALESCE(profile_uuid,''),COALESCE(server_uuid,''),COALESCE(email,''),qq,status,game_verified,admin_version FROM identities WHERE id=? FOR UPDATE`, target).Scan(&gameID, &uuid, &serverUUID, &email, &qq, &status, &verified, &version)
	if err != nil {
		return nil, socialMissing(err)
	}
	if version != in.ExpectedVersion {
		return nil, ErrSocialVersion
	}
	if verified || serverUUID != "" {
		return nil, catalogError(409, "VERIFIED_IDENTITY_PROTECTED", "已验证的游戏身份不能解除或换绑，请单独处理账号申诉。")
	}
	if status == "deleted" || !ValidAdmissionUUID(uuid) || email == "" || gameID == "" {
		return nil, catalogError(409, "PROFILE_NOT_RELEASABLE", "此账号没有可解除的未验证登记，或缺少邮箱恢复入口。")
	}
	if in.ClaimantEmail == email {
		return nil, catalogError(409, "USE_ACCOUNT_RECOVERY", "认领邮箱就是原账号邮箱，请通过找回密码恢复原账号。")
	}
	var claimantID, claimantGameID, claimantStatus string
	var claimantVerified bool
	err = tx.QueryRowContext(ctx, "SELECT id,game_id,status,game_verified FROM identities WHERE email=?", in.ClaimantEmail).Scan(&claimantID, &claimantGameID, &claimantStatus, &claimantVerified)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil && (claimantVerified || claimantGameID != "" || claimantStatus != "active") {
		return nil, catalogError(409, "CLAIMANT_ACCOUNT_NOT_ELIGIBLE", "认领邮箱已有其他游戏身份登记或账号不可用，请先核对工单。")
	}
	var verificationID string
	if in.EvidenceType == "GAME_CODE" {
		verificationID, err = consumeProfileClaimTx(ctx, tx, target, uuid, proof)
		if errors.Is(err, ErrVerification) {
			// Persist wrong-code attempts, but no release has happened yet.
			if commitErr := tx.Commit(); commitErr != nil {
				return nil, commitErr
			}
		}
		if err != nil {
			return nil, err
		}
	}
	caseID := ID("claim_")
	if _, err = tx.ExecContext(ctx, `INSERT INTO identity_profile_resolutions(case_id,actor_id,target_user_id,profile_uuid,previous_game_id,claimant_email,evidence_type,evidence_reference,reason,game_verification_id,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6))`, caseID, proof.ActorID, target, uuid, gameID, in.ClaimantEmail, in.EvidenceType, in.EvidenceReference, in.Reason, nullString(verificationID)); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO identity_profile_reservations(profile_uuid,case_id,claimant_email) VALUES(?,?,?) ON DUPLICATE KEY UPDATE case_id=VALUES(case_id),claimant_email=VALUES(claimant_email),claimed_user_id=NULL,claimed_at=NULL`, uuid, caseID, in.ClaimantEmail); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE identities SET display_name=COALESCE(display_name,game_id),game_id='',profile_uuid=NULL,profile_observed_at=NULL,admin_version=admin_version+1,updated_at=UTC_TIMESTAMP(6) WHERE id=?`, target); err != nil {
		return nil, err
	}
	if gameID != qq && !strings.EqualFold(gameID, email) {
		if _, err = tx.ExecContext(ctx, "DELETE FROM identity_aliases WHERE user_id=? AND alias_key=LOWER(?)", target, gameID); err != nil {
			return nil, err
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE game_verifications SET state='EXPIRED' WHERE user_id=? AND state IN ('ACTIVE','PENDING')", target); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE identity_sessions SET revoked_at=COALESCE(revoked_at,UTC_TIMESTAMP(6)) WHERE user_id=?", target); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,'account.profile-release',?,UTC_TIMESTAMP(6))", proof.ActorID, caseID); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(map[string]any{"caseId": caseID, "userId": target, "released": true, "version": version + 1})
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE social_requests_v2 SET response_json=? WHERE actor_id=? AND action=? AND request_id=?", encoded, proof.ActorID, action, in.ClientRequestID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return encoded, nil
}

func consumeProfileClaimTx(ctx context.Context, tx *sql.Tx, target, uuid string, proof ProfileReleaseProof) (string, error) {
	var v GameVerification
	var attempts int
	err := tx.QueryRowContext(ctx, `SELECT verification_id,code_hash,user_id,player_uuid,state,expires_at,attempts FROM game_verifications WHERE token_hash=? AND purpose='profile_claim' FOR UPDATE`, proof.GameTokenHash).Scan(&v.ID, &v.CodeHash, &v.UserID, &v.PlayerUUID, &v.State, &v.ExpiresAt, &attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrVerification
	}
	if err != nil {
		return "", err
	}
	if v.UserID != target || v.PlayerUUID != uuid || v.State != "ACTIVE" || attempts >= 5 || !v.ExpiresAt.After(time.Now().UTC()) {
		return "", ErrVerification
	}
	if subtle.ConstantTimeCompare([]byte(v.CodeHash), []byte(proof.GameCodeHash)) != 1 {
		if _, err = tx.ExecContext(ctx, "UPDATE game_verifications SET attempts=attempts+1 WHERE verification_id=?", v.ID); err != nil {
			return "", err
		}
		return "", ErrVerification
	}
	return v.ID, consumeVerification(ctx, tx, v)
}

func profileReservationTx(ctx context.Context, tx *sql.Tx, uuid, email, userID string) error {
	if uuid == "" {
		return nil
	}
	var reservedEmail, caseID string
	err := tx.QueryRowContext(ctx, "SELECT claimant_email,case_id FROM identity_profile_reservations WHERE profile_uuid=? AND claimed_user_id IS NULL FOR UPDATE", uuid).Scan(&reservedEmail, &caseID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if reservedEmail != email {
		return catalogError(409, "GAME_PROFILE_RESERVED", "此游戏身份正在处理归属，请使用工单中确认的邮箱继续，或联系支持人员。")
	}
	if userID != "" {
		_, err = tx.ExecContext(ctx, "UPDATE identity_profile_reservations SET claimed_user_id=?,claimed_at=UTC_TIMESTAMP(6) WHERE profile_uuid=? AND claimed_user_id IS NULL", userID, uuid)
		if err == nil {
			_, err = tx.ExecContext(ctx, "UPDATE identity_profile_resolutions SET claimed_user_id=?,claimed_at=UTC_TIMESTAMP(6) WHERE case_id=?", userID, caseID)
		}
	}
	return err
}

func (s *Store) RegisterReleasedProfile(ctx context.Context, session Session, key string, p AdmissionProfile) (json.RawMessage, error) {
	if !ValidAdmissionUUID(p.UUID) || !ValidAdmissionName(p.Name) {
		return nil, ErrSocialInvalid
	}
	return s.socialMutate(ctx, session.User.ID, "account.profile-register", key, map[string]string{"gameId": strings.ToLower(p.Name)}, func(tx *sql.Tx) (any, error) {
		var gameID, email, uuid string
		var verified bool
		if err := tx.QueryRowContext(ctx, "SELECT game_id,COALESCE(email,''),COALESCE(profile_uuid,''),game_verified FROM identities WHERE id=? FOR UPDATE", session.User.ID).Scan(&gameID, &email, &uuid, &verified); err != nil {
			return nil, err
		}
		if verified || gameID != "" || uuid != "" || email == "" {
			return nil, catalogError(409, "PROFILE_ALREADY_REGISTERED", "已有游戏身份登记，不能在此更换绑定。")
		}
		var current int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM identity_sessions WHERE token_hash=? AND user_id=? AND revoked_at IS NULL AND (expires_at>UTC_TIMESTAMP(6) OR (expires_at IS NULL AND client_kind='app'))`, session.TokenHash, session.User.ID).Scan(&current); err != nil {
			return nil, err
		}
		if current != 1 {
			return nil, ErrUnauthorized
		}
		if err := profileReservationTx(ctx, tx, p.UUID, email, session.User.ID); err != nil {
			return nil, err
		}
		var occupied int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM identities WHERE id<>? AND status<>'deleted' AND (profile_uuid=? OR server_uuid=? OR LOWER(game_id)=LOWER(?))", session.User.ID, p.UUID, p.UUID, p.Name).Scan(&occupied); err != nil {
			return nil, err
		}
		if occupied > 0 {
			return nil, ErrNameConflict
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO identity_aliases(alias_key,user_id) VALUES(LOWER(?),?)", p.Name, session.User.ID); err != nil {
			if duplicate(err) {
				return nil, ErrNameConflict
			}
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE identities SET game_id=?,profile_uuid=?,profile_observed_at=UTC_TIMESTAMP(6),admin_version=admin_version+1,updated_at=UTC_TIMESTAMP(6) WHERE id=?", p.Name, p.UUID, session.User.ID); err != nil {
			if duplicate(err) {
				return nil, ErrNameConflict
			}
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,'account.profile-register',?,UTC_TIMESTAMP(6))", session.User.ID, session.User.ID); err != nil {
			return nil, err
		}
		return map[string]any{"gameId": p.Name, "registered": true, "gameVerified": false}, nil
	})
}

func (s *Store) ProfileResolutions(ctx context.Context, target string) ([]map[string]any, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT c.case_id,c.previous_game_id,c.claimant_email,c.evidence_type,c.evidence_reference,c.reason,c.actor_id,c.created_at,COALESCE(c.claimed_user_id,''),c.claimed_at,COALESCE(r.claimant_email,c.claimant_email),r.case_id IS NOT NULL AND r.claimed_user_id IS NULL FROM identity_profile_resolutions c LEFT JOIN identity_profile_reservations r ON r.case_id=c.case_id WHERE c.target_user_id=? ORDER BY c.created_at DESC LIMIT 50`, target)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var caseID, name, email, evidence, reference, reason, actor, claimed, currentEmail string
		var pending bool
		var created time.Time
		var claimedAt sql.NullTime
		if err = rows.Scan(&caseID, &name, &email, &evidence, &reference, &reason, &actor, &created, &claimed, &claimedAt, &currentEmail, &pending); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{"caseId": caseID, "previousGameId": name, "claimantEmail": email, "currentClaimantEmail": currentEmail, "pending": pending, "evidenceType": evidence, "evidenceReference": reference, "reason": reason, "actorId": actor, "createdAt": created, "claimedUserId": claimed, "claimedAt": commerceTime(claimedAt)})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	byCase := map[string]map[string]any{}
	for _, item := range items {
		item["emailCorrections"] = []map[string]any{}
		byCase[item["caseId"].(string)] = item
	}
	edits, err := s.DB.QueryContext(ctx, `SELECT e.edit_id,e.case_id,e.previous_email,e.next_email,e.reason,e.actor_id,e.created_at FROM identity_profile_reservation_edits e JOIN identity_profile_resolutions c ON c.case_id=e.case_id WHERE c.target_user_id=? ORDER BY e.created_at DESC LIMIT 200`, target)
	if err != nil {
		return nil, err
	}
	defer edits.Close()
	for edits.Next() {
		var id, caseID, previous, next, reason, actor string
		var created time.Time
		if err = edits.Scan(&id, &caseID, &previous, &next, &reason, &actor, &created); err != nil {
			return nil, err
		}
		if item := byCase[caseID]; item != nil {
			item["emailCorrections"] = append(item["emailCorrections"].([]map[string]any), map[string]any{"editId": id, "previousEmail": previous, "nextEmail": next, "reason": reason, "actorId": actor, "createdAt": created})
		}
	}
	return items, edits.Err()
}

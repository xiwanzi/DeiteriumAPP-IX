package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
)

// Fail before sending an email when a claim explicitly reserves a profile for
// a different mailbox. The registration transaction repeats this check under lock.
func (s *Store) CheckProfileReservation(ctx context.Context, uuid, email string) error {
	var reserved string
	err := s.DB.QueryRowContext(ctx, "SELECT claimant_email FROM identity_profile_reservations WHERE profile_uuid=? AND claimed_user_id IS NULL", uuid).Scan(&reserved)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if reserved != email {
		return catalogError(409, "GAME_PROFILE_RESERVED", "此游戏身份正在处理归属，请使用工单中确认的邮箱继续，或联系支持人员。")
	}
	return nil
}

func (s *Store) RegisteredProfileResult(ctx context.Context, user, key, name string) (json.RawMessage, bool, error) {
	var fingerprint string
	var result sql.NullString
	err := s.DB.QueryRowContext(ctx, "SELECT fingerprint,response_json FROM social_requests_v2 WHERE actor_id=? AND action='account.profile-register' AND request_id=?", user, key).Scan(&fingerprint, &result)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	want := Digest([]byte(catalogJSON(map[string]string{"gameId": strings.ToLower(strings.TrimSpace(name))})))
	if fingerprint != want {
		return nil, false, ErrConflict
	}
	return json.RawMessage(result.String), result.Valid, nil
}

type ProfileReservationCorrection struct {
	ClientRequestID string `json:"clientRequestId"`
	ExpectedEmail   string `json:"expectedEmail"`
	ClaimantEmail   string `json:"claimantEmail"`
	Reason          string `json:"reason"`
}

// Correct a typo before a claimant registers. Completed registrations cannot be
// redirected: changing an email here never moves an existing platform account.
func (s *Store) CorrectProfileReservation(ctx context.Context, caseID string, in ProfileReservationCorrection, proof AccountDeletionProof) (json.RawMessage, error) {
	var err error
	if in.ClaimantEmail, err = NormalizeEmail(in.ClaimantEmail); err != nil {
		return nil, err
	}
	if in.ExpectedEmail, err = NormalizeEmail(in.ExpectedEmail); err != nil {
		return nil, err
	}
	in.Reason = strings.TrimSpace(in.Reason)
	if !ValidSocialID(caseID) || !ValidSocialText(in.Reason, 3, 500) {
		return nil, ErrSocialInvalid
	}
	return s.socialMutate(ctx, proof.ActorID, "account.claim-email:"+caseID, in.ClientRequestID, in, func(tx *sql.Tx) (any, error) {
		if err := requireAdminSessionProofTx(ctx, tx, proof); err != nil {
			return nil, err
		}
		var email, target, originalEmail string
		var claimed sql.NullString
		err := tx.QueryRowContext(ctx, `SELECT r.claimant_email,r.claimed_user_id,c.target_user_id FROM identity_profile_reservations r JOIN identity_profile_resolutions c ON c.case_id=r.case_id WHERE r.case_id=? FOR UPDATE`, caseID).Scan(&email, &claimed, &target)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSocialNotFound
		}
		if err != nil {
			return nil, err
		}
		if err = tx.QueryRowContext(ctx, "SELECT COALESCE(email,'') FROM identities WHERE id=?", target).Scan(&originalEmail); err != nil {
			return nil, err
		}
		if claimed.Valid {
			return nil, catalogError(409, "CLAIM_ALREADY_REGISTERED", "持有人已注册，不能修改归属邮箱。")
		}
		if email != in.ExpectedEmail {
			return nil, ErrSocialVersion
		}
		if in.ClaimantEmail == originalEmail {
			return nil, catalogError(409, "USE_ACCOUNT_RECOVERY", "不能将已解除的身份预留回原登记者邮箱。")
		}
		var occupied int
		if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM identities WHERE email=? AND (game_verified=TRUE OR game_id<>'' OR status<>'active')", in.ClaimantEmail).Scan(&occupied); err != nil {
			return nil, err
		}
		if occupied > 0 {
			return nil, catalogError(409, "CLAIMANT_ACCOUNT_NOT_ELIGIBLE", "此邮箱已有其他游戏身份登记或账号不可用，请先核对工单。")
		}
		if email != in.ClaimantEmail {
			if _, err = tx.ExecContext(ctx, "UPDATE identity_profile_reservations SET claimant_email=? WHERE case_id=?", in.ClaimantEmail, caseID); err != nil {
				return nil, err
			}
			editID := ID("cedit_")
			if _, err = tx.ExecContext(ctx, "INSERT INTO identity_profile_reservation_edits(edit_id,case_id,actor_id,previous_email,next_email,reason,created_at) VALUES(?,?,?,?,?,?,UTC_TIMESTAMP(6))", editID, caseID, proof.ActorID, email, in.ClaimantEmail, in.Reason); err != nil {
				return nil, err
			}
			if _, err = tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,'account.claim-email',?,UTC_TIMESTAMP(6))", proof.ActorID, editID); err != nil {
				return nil, err
			}
		}
		return map[string]any{"caseId": caseID, "claimantEmail": in.ClaimantEmail}, nil
	})
}

package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/notify"
)

type AdmissionEmailPayload struct {
	TemplateVersion string `json:"templateVersion"`
	UUID            string `json:"uuid"`
	GameID          string `json:"gameId"`
	Decision        string `json:"decision"`
	Reason          string `json:"reason"`
	ReviewVersion   int64  `json:"reviewVersion"`
	EntryVersion    int64  `json:"entryVersion"`
}

func enqueueAdmissionEmail(ctx context.Context, tx *sql.Tx, application, qq string, payload AdmissionEmailPayload) (string, error) {
	var settingsJSON string
	err := tx.QueryRowContext(ctx, "SELECT settings_json FROM admin_email_settings_v204 WHERE id=1").Scan(&settingsJSON)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	var settings notify.Settings
	if settingsJSON != "" {
		if err = json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
			return "", err
		}
	}
	state, reason := "PENDING", ""
	if !settings.AdmissionReviewEnabled {
		state, reason = "SKIPPED", "审核时白名单邮件提醒未开启。"
	}
	if !admissionQQ.MatchString(qq) {
		return "", ErrSocialInvalid
	}
	payload.TemplateVersion = notify.AdmissionTemplateVersion
	if payload.Decision == "APPROVED" {
		payload.Reason = "" // Approval notes are private to administrators.
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO admin_email_outbox_v204(event_id,application_id,recipient,payload_json,case_version,case_state,status,last_error,next_attempt_at,created_at) VALUES(?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, "admission:"+application, application, qq+"@qq.com", raw, payload.ReviewVersion, payload.Decision, state, reason)
	return state, err
}

func (s *Store) AdmissionEmailCurrent(ctx context.Context, event EmailEventV204, payload AdmissionEmailPayload) (bool, error) {
	if event.ApplicationID == nil {
		return true, nil // Explicit template tests have no real application.
	}
	var status, uuid, gameID, qq, reason string
	var version int64
	var created time.Time
	err := s.DB.QueryRowContext(ctx, `SELECT status,server_uuid,game_id,qq,reason,version,created_at FROM admission_applications WHERE application_id=?`, *event.ApplicationID).Scan(&status, &uuid, &gameID, &qq, &reason, &version, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if status != payload.Decision || version != payload.ReviewVersion || uuid != payload.UUID || gameID != payload.GameID || event.Recipient != qq+"@qq.com" || (status == "REJECTED" && reason != payload.Reason) {
		return false, nil
	}
	var newer int
	if err = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM admission_applications WHERE server_uuid=? AND created_at>?`, uuid, created).Scan(&newer); err != nil {
		return false, err
	}
	if newer > 0 {
		return false, nil
	}
	var entryStatus, entryApplication string
	var entryVersion int64
	err = s.DB.QueryRowContext(ctx, `SELECT status,application_id,version FROM admission_entries WHERE server_uuid=?`, uuid).Scan(&entryStatus, &entryApplication, &entryVersion)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	if payload.Decision == "APPROVED" {
		return err == nil && entryStatus == "ACTIVE" && entryVersion == payload.EntryVersion && entryApplication == *event.ApplicationID, nil
	}
	return entryStatus != "ACTIVE", nil
}

func (s *Store) CancelEmailV204(ctx context.Context, event EmailEventV204, reason string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE admin_email_outbox_v204 SET status='CANCELLED',lease_until=NULL,last_error=? WHERE event_id=? AND attempts=? AND status='SENDING'`, reason, event.EventID, event.Attempts)
	return err
}

func (s *Store) BindEmailRecipientsV204(ctx context.Context, event EmailEventV204, recipients []string) error {
	r, err := s.DB.ExecContext(ctx, `UPDATE admin_email_outbox_v204 SET recipient=? WHERE event_id=? AND attempts=? AND status='SENDING' AND recipient=''`, strings.Join(recipients, ", "), event.EventID, event.Attempts)
	if err != nil {
		return err
	}
	if n, err := r.RowsAffected(); err != nil || n != 1 {
		return ErrSocialVersion
	}
	return nil
}

func (s *Store) EnqueueAdmissionEmailPreview(ctx context.Context, actor, key, decision string) (json.RawMessage, error) {
	if decision != "APPROVED" && decision != "REJECTED" {
		return nil, ErrSocialInvalid
	}
	return s.socialMutate(ctx, actor, "email.test:"+decision, key, key, func(tx *sql.Tx) (any, error) {
		if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
			return nil, err
		}
		var settingsJSON, gameID string
		if err := tx.QueryRowContext(ctx, `SELECT settings_json FROM admin_email_settings_v204 WHERE id=1 FOR UPDATE`).Scan(&settingsJSON); err != nil {
			return nil, err
		}
		var settings notify.Settings
		if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
			return nil, err
		}
		if err := tx.QueryRowContext(ctx, `SELECT game_id FROM identities WHERE id=?`, actor).Scan(&gameID); err != nil {
			return nil, err
		}
		state := "TEST_" + decision
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM admin_email_outbox_v204 WHERE application_id IS NULL AND case_id IS NULL AND case_state=? AND created_at>DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 MINUTE)`, state).Scan(&count); err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, ErrRateLimited
		}
		payload := AdmissionEmailPayload{TemplateVersion: notify.AdmissionTemplateVersion, GameID: gameID, Decision: decision, Reason: "申请中的玩家信息与核验结果不一致，请核对正版玩家 ID 后重新提交。"}
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		id := ID("smtp_test_")
		_, err = tx.ExecContext(ctx, `INSERT INTO admin_email_outbox_v204(event_id,recipient,payload_json,case_version,case_state,next_attempt_at,created_at) VALUES(?,?,?,0,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, id, strings.Join(settings.Recipients, ", "), raw, state)
		return map[string]any{"eventId": id, "status": "PENDING"}, err
	})
}

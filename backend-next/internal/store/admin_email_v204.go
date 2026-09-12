package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/notify"
)

type EmailSettingsV204 struct {
	notify.Settings
	PasswordCipher string `json:"-"`
}
type EmailEventV204 struct {
	EventID       string     `json:"eventId"`
	CaseID        *string    `json:"caseId"`
	CaseVersion   int64      `json:"caseVersion"`
	CaseState     string     `json:"caseState"`
	Status        string     `json:"status"`
	Attempts      int        `json:"attempts"`
	NextAttemptAt time.Time  `json:"nextAttemptAt"`
	LastError     string     `json:"lastError"`
	CreatedAt     time.Time  `json:"createdAt"`
	SentAt        *time.Time `json:"sentAt"`
}

func (s *Store) EmailSettingsV204(ctx context.Context) (EmailSettingsV204, error) {
	var result EmailSettingsV204
	var raw string
	err := s.DB.QueryRowContext(ctx, `SELECT settings_json,password_cipher,version FROM admin_email_settings_v204 WHERE id=1`).Scan(&raw, &result.PasswordCipher, &result.Version)
	if errors.Is(err, sql.ErrNoRows) {
		result.Security = "STARTTLS"
		result.Port = 587
		result.Recipients = []string{}
		return result, nil
	}
	if err != nil {
		return result, err
	}
	version := result.Version
	err = json.Unmarshal([]byte(raw), &result.Settings)
	result.Version = version
	result.PasswordConfigured = result.PasswordCipher != ""
	return result, err
}
func (s *Store) SaveEmailSettingsV204(ctx context.Context, actor, key string, expected int64, settings notify.Settings, password *string, encryptionKey []byte) (json.RawMessage, error) {
	if expected < 0 || settings.Validate() != nil {
		return nil, ErrSocialInvalid
	}
	return s.socialMutate(ctx, actor, "email.settings", key, struct {
		Expected int64
		Settings notify.Settings
		Password *string
	}{expected, settings, password}, func(tx *sql.Tx) (any, error) {
		if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO admin_email_settings_v204 VALUES(1,'{}','',0,UTC_TIMESTAMP(6))`); err != nil {
			return nil, err
		}
		var version int64
		var cipher string
		if err := tx.QueryRowContext(ctx, `SELECT version,password_cipher FROM admin_email_settings_v204 WHERE id=1 FOR UPDATE`).Scan(&version, &cipher); err != nil {
			return nil, err
		}
		if expected != version {
			return nil, ErrSocialVersion
		}
		if password != nil {
			cipher = ""
			if *password != "" {
				var err error
				cipher, err = notify.Seal(encryptionKey, *password)
				if err != nil {
					return nil, err
				}
			}
		}
		if settings.Username != "" && cipher == "" {
			return nil, ErrSocialInvalid
		}
		settings.Version = version + 1
		settings.PasswordConfigured = cipher != ""
		raw, _ := json.Marshal(settings)
		if _, err := tx.ExecContext(ctx, `UPDATE admin_email_settings_v204 SET settings_json=?,password_cipher=?,version=?,updated_at=UTC_TIMESTAMP(6) WHERE id=1`, raw, cipher, settings.Version); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,'email.settings','smtp',UTC_TIMESTAMP(6))`, actor); err != nil {
			return nil, err
		}
		return settings, nil
	})
}
func enqueueCaseEmailV204(ctx context.Context, tx *sql.Tx, id string) error {
	_, err := tx.ExecContext(ctx, `INSERT IGNORE INTO admin_email_outbox_v204(event_id,case_id,case_version,case_state,next_attempt_at,created_at) SELECT CONCAT('case:',case_id,':',version),case_id,version,state,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6) FROM commerce_interventions_v2 WHERE case_id=?`, id)
	return err
}
func (s *Store) EnqueueTestEmailV204(ctx context.Context, actor, key string) (json.RawMessage, error) {
	return s.socialMutate(ctx, actor, "email.test", key, key, func(tx *sql.Tx) (any, error) {
		var version int64
		if err := tx.QueryRowContext(ctx, `SELECT version FROM admin_email_settings_v204 WHERE id=1 FOR UPDATE`).Scan(&version); err != nil {
			return nil, err
		}
		// A test message is explicit and rate limited; it never contains case evidence.
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM admin_email_outbox_v204 WHERE case_id IS NULL AND created_at>DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 MINUTE)`).Scan(&count); err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, ErrRateLimited
		}
		id := ID("smtp_test_")
		_, err := tx.ExecContext(ctx, `INSERT INTO admin_email_outbox_v204(event_id,case_version,case_state,next_attempt_at,created_at) VALUES(?,0,'TEST',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, id)
		return map[string]any{"eventId": id, "status": "PENDING"}, err
	})
}

const emailEventSelectV204 = `SELECT event_id,case_id,case_version,case_state,status,attempts,next_attempt_at,last_error,created_at,sent_at FROM admin_email_outbox_v204`

func scanEmailEventV204(row catalogScanner) (e EmailEventV204, err error) {
	err = row.Scan(&e.EventID, &e.CaseID, &e.CaseVersion, &e.CaseState, &e.Status, &e.Attempts, &e.NextAttemptAt, &e.LastError, &e.CreatedAt, &e.SentAt)
	return
}
func (s *Store) EmailStatusV204(ctx context.Context) (map[string]any, error) {
	var pending, failed int
	err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(status<>'SENT'),0),COALESCE(SUM(status='RETRY'),0) FROM admin_email_outbox_v204`).Scan(&pending, &failed)
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, emailEventSelectV204+` ORDER BY created_at DESC,event_id DESC LIMIT 20`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []EmailEventV204{}
	for rows.Next() {
		e, err := scanEmailEventV204(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return map[string]any{"pending": pending, "retrying": failed, "events": events}, rows.Err()
}
func (s *Store) ClaimEmailV204(ctx context.Context) (EmailEventV204, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return EmailEventV204{}, err
	}
	defer tx.Rollback()
	// The runtime has one email worker; ordinary row locking also supports MariaDB 10.5.
	event, err := scanEmailEventV204(tx.QueryRowContext(ctx, emailEventSelectV204+` WHERE (status IN ('PENDING','RETRY') AND next_attempt_at<=UTC_TIMESTAMP(6)) OR (status='SENDING' AND lease_until<UTC_TIMESTAMP(6)) ORDER BY created_at,event_id LIMIT 1 FOR UPDATE`))
	if err != nil {
		return event, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE admin_email_outbox_v204 SET status='SENDING',attempts=attempts+1,lease_until=DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 60 SECOND) WHERE event_id=?`, event.EventID)
	if err != nil {
		return event, err
	}
	event.Attempts++
	err = tx.Commit()
	return event, err
}
func (s *Store) FinishEmailV204(ctx context.Context, event EmailEventV204, sendError error) error {
	if sendError == nil {
		_, err := s.DB.ExecContext(ctx, `UPDATE admin_email_outbox_v204 SET status='SENT',sent_at=UTC_TIMESTAMP(6),lease_until=NULL,last_error='' WHERE event_id=? AND attempts=? AND status='SENDING'`, event.EventID, event.Attempts)
		return err
	}
	delay := min(3600, 15*(1<<min(event.Attempts, 8)))
	message := sendError.Error()
	if len([]rune(message)) > 300 {
		message = string([]rune(message)[:300])
	}
	_, err := s.DB.ExecContext(ctx, `UPDATE admin_email_outbox_v204 SET status='RETRY',next_attempt_at=DATE_ADD(UTC_TIMESTAMP(6),INTERVAL `+strconv.Itoa(delay)+` SECOND),lease_until=NULL,last_error=? WHERE event_id=? AND attempts=? AND status='SENDING'`, message, event.EventID, event.Attempts)
	return err
}

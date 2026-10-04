package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/notify"
)

var ErrMailSkipped = errors.New("expired mail skipped")

func NormalizeEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value || len(value) > 254 || !strings.Contains(value, "@") || strings.IndexFunc(value, func(r rune) bool { return r > 127 || unicode.IsControl(r) }) >= 0 {
		return "", ErrSocialInvalid
	}
	return value, nil
}

func (s *Store) MailChannel(ctx context.Context, channel string) (EmailSettingsV204, error) {
	var result EmailSettingsV204
	var raw string
	err := s.DB.QueryRowContext(ctx, "SELECT settings_json,password_cipher,version FROM mail_channels WHERE channel_id=?", channel).Scan(&raw, &result.PasswordCipher, &result.Version)
	if errors.Is(err, sql.ErrNoRows) {
		result.Security = "TLS"
		result.Port = 465
		result.From = map[string]string{"DM": "noreply@notify.deuteriumix.com", "ENTERPRISE": "support@deuteriumix.com", "SYSTEM": "go@deuteriumix.com"}[channel]
		result.Username = result.From
		result.Recipients = []string{result.From}
		result.Host = "smtp.qiye.aliyun.com"
		if channel == "DM" {
			result.Host = "smtpdm.aliyun.com"
		}
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

// Preserve existing review/operations notifications until DM is explicitly configured.
// Once saved, disabling DM pauses delivery; it must not silently switch senders.
func (s *Store) NotificationMailChannel(ctx context.Context) (EmailSettingsV204, error) {
	settings, err := s.MailChannel(ctx, "DM")
	if err == nil && settings.Version == 0 {
		return s.EmailSettingsV204(ctx)
	}
	return settings, err
}

func (s *Store) SaveMailNotifications(ctx context.Context, actor, request string, expected int64, review bool, recipients []string) (json.RawMessage, error) {
	if expected < 0 || len(recipients) > 10 {
		return nil, ErrSocialInvalid
	}
	seen := map[string]bool{}
	for _, email := range recipients {
		normalized, err := NormalizeEmail(email)
		if err != nil || normalized != email || seen[email] {
			return nil, ErrSocialInvalid
		}
		seen[email] = true
	}
	return s.socialMutate(ctx, actor, "mail.notifications", request, struct {
		Version    int64
		Review     bool
		Recipients []string
	}{expected, review, recipients}, func(tx *sql.Tx) (any, error) {
		if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, "INSERT IGNORE INTO admin_email_settings_v204 VALUES(1,'{}','',0,UTC_TIMESTAMP(6))"); err != nil {
			return nil, err
		}
		var raw string
		var version int64
		if err := tx.QueryRowContext(ctx, "SELECT settings_json,version FROM admin_email_settings_v204 WHERE id=1 FOR UPDATE").Scan(&raw, &version); err != nil {
			return nil, err
		}
		if version != expected {
			return nil, ErrSocialVersion
		}
		var settings notify.Settings
		if err := json.Unmarshal([]byte(raw), &settings); err != nil {
			return nil, err
		}
		settings.AdmissionReviewEnabled = review
		settings.Recipients = recipients
		settings.Version = version + 1
		encoded, _ := json.Marshal(settings)
		_, err := tx.ExecContext(ctx, "UPDATE admin_email_settings_v204 SET settings_json=?,version=version+1,updated_at=UTC_TIMESTAMP(6) WHERE id=1", encoded)
		return map[string]any{"version": version + 1, "admissionReviewEnabled": review, "recipients": recipients}, err
	})
}

func (s *Store) SaveMailChannel(ctx context.Context, actor, channel, key string, expected int64, settings notify.Settings, password *string, encryptionKey []byte) (json.RawMessage, error) {
	if !catalogEnum(channel, "DM", "ENTERPRISE", "SYSTEM") || settings.Validate() != nil || expected < 0 {
		return nil, ErrSocialInvalid
	}
	if password != nil && (len(*password) > 2048 || strings.ContainsRune(*password, 0)) {
		return nil, ErrSocialInvalid
	}
	return s.socialMutate(ctx, actor, "mail.channel:"+channel, key, struct {
		Expected int64
		Settings notify.Settings
		Password *string
	}{expected, settings, password}, func(tx *sql.Tx) (any, error) {
		if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, "INSERT IGNORE INTO mail_channels VALUES(?,'{}','',0,UTC_TIMESTAMP(6))", channel); err != nil {
			return nil, err
		}
		var version int64
		var cipher string
		if err := tx.QueryRowContext(ctx, "SELECT version,password_cipher FROM mail_channels WHERE channel_id=? FOR UPDATE", channel).Scan(&version, &cipher); err != nil {
			return nil, err
		}
		if version != expected {
			return nil, ErrSocialVersion
		}
		if password != nil {
			var err error
			cipher, err = notify.Seal(encryptionKey, *password)
			if err != nil {
				return nil, err
			}
		}
		if settings.Username != "" && cipher == "" {
			return nil, ErrSocialInvalid
		}
		settings.Version = version + 1
		settings.PasswordConfigured = cipher != ""
		raw, _ := json.Marshal(settings)
		if _, err := tx.ExecContext(ctx, "UPDATE mail_channels SET settings_json=?,password_cipher=?,version=?,updated_at=UTC_TIMESTAMP(6) WHERE channel_id=?", raw, cipher, settings.Version, channel); err != nil {
			return nil, err
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,'mail.channel',?,UTC_TIMESTAMP(6))", actor, channel)
		return settings, err
	})
}

type SystemMailEvent struct {
	ReplyTo       string     `json:"replyTo,omitempty"`
	InReplyTo     string     `json:"-"`
	HeadersCipher string     `json:"-"`
	EventID       string     `json:"eventId"`
	Channel       string     `json:"channel"`
	Kind          string     `json:"kind"`
	Recipient     string     `json:"recipient"`
	Subject       string     `json:"subject"`
	Reference     string     `json:"referenceId"`
	Status        string     `json:"status"`
	Attempts      int        `json:"attempts"`
	LastError     string     `json:"lastError"`
	CreatedAt     time.Time  `json:"createdAt"`
	SentAt        *time.Time `json:"sentAt"`
	Cipher        string     `json:"-"`
}

const mailEventSelect = `SELECT event_id,channel_id,kind,recipient,subject,COALESCE(reference_id,''),status,attempts,last_error,created_at,sent_at,payload_cipher,reply_to,in_reply_to,COALESCE(headers_cipher,'') FROM mail_outbox`

func scanSystemMailEvent(row catalogScanner) (e SystemMailEvent, err error) {
	err = row.Scan(&e.EventID, &e.Channel, &e.Kind, &e.Recipient, &e.Subject, &e.Reference, &e.Status, &e.Attempts, &e.LastError, &e.CreatedAt, &e.SentAt, &e.Cipher, &e.ReplyTo, &e.InReplyTo, &e.HeadersCipher)
	return
}

func enqueueMailTx(ctx context.Context, tx *sql.Tx, channel, kind, recipient, subject, body, reference, actor string, key []byte) (string, error) {
	cipher, err := notify.Seal(key, body)
	if err != nil {
		return "", err
	}
	id := ID("mail_")
	_, err = tx.ExecContext(ctx, `INSERT INTO mail_outbox(event_id,channel_id,kind,recipient,subject,payload_cipher,reference_id,actor_id,next_attempt_at,created_at) VALUES(?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, id, channel, kind, recipient, subject, cipher, nullString(reference), nullString(actor))
	return id, err
}

func (s *Store) QueueManualMail(ctx context.Context, actor, request, channel, recipient, subject, body string, key []byte) (json.RawMessage, error) {
	recipient, err := NormalizeEmail(recipient)
	if err != nil || !catalogEnum(channel, "DM", "ENTERPRISE") || !ValidSocialText(subject, 1, 160) || strings.ContainsAny(subject, "\r\n") || !ValidSocialText(body, 1, 15000) {
		return nil, ErrSocialInvalid
	}
	return s.socialMutate(ctx, actor, "mail.compose", request, []string{channel, recipient, subject, body}, func(tx *sql.Tx) (any, error) {
		if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
			return nil, err
		}
		var raw string
		if err := tx.QueryRowContext(ctx, "SELECT settings_json FROM mail_channels WHERE channel_id=?", channel).Scan(&raw); err != nil {
			return nil, err
		}
		var settings notify.Settings
		if json.Unmarshal([]byte(raw), &settings) != nil || !settings.Enabled {
			return nil, catalogError(409, "MAIL_CHANNEL_DISABLED", "请先配置并启用发信通道。")
		}
		id, err := enqueueMailTx(ctx, tx, channel, "MANUAL", recipient, subject, body, "", actor, key)
		return map[string]string{"eventId": id, "status": "PENDING"}, err
	})
}

func (s *Store) SystemMailEvents(ctx context.Context) ([]SystemMailEvent, error) {
	rows, err := s.DB.QueryContext(ctx, mailEventSelect+" ORDER BY created_at DESC,event_id DESC LIMIT 100")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SystemMailEvent{}
	for rows.Next() {
		e, err := scanSystemMailEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Ticket replies retain the public signature recorded when the reply was written.
func (s *Store) SystemMailAuthorName(ctx context.Context, eventID string) (string, error) {
	var name string
	err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(
	 (SELECT m.author_name FROM support_messages m WHERE m.ticket_id=e.reference_id AND m.mail_message_id=CONCAT(e.event_id,'@deuterium-notifications') AND m.direction='OUTBOUND' AND m.visibility='PUBLIC' LIMIT 1),
	 NULLIF(i.display_name,''),NULLIF(i.game_id,''),'')
	 FROM mail_outbox e LEFT JOIN identities i ON i.id=e.actor_id AND i.status='active' WHERE e.event_id=?`, eventID).Scan(&name)
	return name, err
}

func (s *Store) ClaimMail(ctx context.Context) (SystemMailEvent, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return SystemMailEvent{}, err
	}
	defer tx.Rollback()
	e, err := scanSystemMailEvent(tx.QueryRowContext(ctx, mailEventSelect+` WHERE ((status IN ('PENDING','RETRY') AND next_attempt_at<=UTC_TIMESTAMP(6)) OR (status='SENDING' AND lease_until<UTC_TIMESTAMP(6))) AND EXISTS(SELECT 1 FROM mail_channels c WHERE c.channel_id=mail_outbox.channel_id AND JSON_UNQUOTE(JSON_EXTRACT(c.settings_json,'$.enabled'))='true') ORDER BY CASE WHEN kind LIKE 'CODE_%' THEN 0 WHEN kind='TICKET_CREATED' THEN 1 ELSE 2 END,created_at,event_id LIMIT 1 FOR UPDATE`))
	if err != nil {
		return e, err
	}
	if strings.HasPrefix(e.Kind, "CODE_") {
		var active bool
		if err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM email_verifications WHERE verification_id=? AND state='ACTIVE' AND expires_at>UTC_TIMESTAMP(6))", e.Reference).Scan(&active); err != nil {
			return e, err
		}
		if !active {
			_, err = tx.ExecContext(ctx, "UPDATE mail_outbox SET status='CANCELLED',payload_cipher='' WHERE event_id=?", e.EventID)
			if err == nil {
				err = tx.Commit()
			}
			if err != nil {
				return e, err
			}
			return e, ErrMailSkipped
		}
	}
	_, err = tx.ExecContext(ctx, "UPDATE mail_outbox SET status='SENDING',attempts=attempts+1,lease_until=DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 60 SECOND) WHERE event_id=?", e.EventID)
	if err != nil {
		return e, err
	}
	e.Attempts++
	err = tx.Commit()
	return e, err
}

func (s *Store) RetrySystemMail(ctx context.Context, actor, event, ticket, request string) (json.RawMessage, error) {
	return s.socialMutate(ctx, actor, "mail.retry:"+event, request, ticket, func(tx *sql.Tx) (any, error) {
		if ticket == "" {
			if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
				return nil, err
			}
		} else if err := requireSupportTx(ctx, tx, actor); err != nil {
			return nil, err
		}
		var state, kind, reference string
		if err := tx.QueryRowContext(ctx, "SELECT status,kind,COALESCE(reference_id,'') FROM mail_outbox WHERE event_id=? FOR UPDATE", event).Scan(&state, &kind, &reference); err != nil {
			return nil, socialMissing(err)
		}
		if ticket != "" && (reference != ticket || !catalogEnum(kind, "TICKET_CREATED", "TICKET_REPLY")) {
			return nil, ErrSocialNotFound
		}
		if state != "FAILED" || strings.HasPrefix(kind, "CODE_") {
			return nil, catalogError(409, "MAIL_NOT_RETRYABLE", "仅可重新发送已停止重试的业务邮件；验证码请重新获取。")
		}
		if _, err := tx.ExecContext(ctx, "UPDATE mail_outbox SET status='PENDING',attempts=0,next_attempt_at=UTC_TIMESTAMP(6),last_error='',lease_until=NULL WHERE event_id=?", event); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,'mail.retry',?,UTC_TIMESTAMP(6))", actor, event); err != nil {
			return nil, err
		}
		return map[string]string{"eventId": event, "status": "PENDING"}, nil
	})
}
func (s *Store) FinishMail(ctx context.Context, e SystemMailEvent, sendErr error) error {
	if sendErr == nil {
		_, err := s.DB.ExecContext(ctx, "UPDATE mail_outbox SET status='SENT',sent_at=UTC_TIMESTAMP(6),last_error='',lease_until=NULL,payload_cipher=IF(kind LIKE 'CODE_%','',payload_cipher) WHERE event_id=? AND attempts=? AND status='SENDING'", e.EventID, e.Attempts)
		return err
	}
	message := sendErr.Error()
	if len([]rune(message)) > 300 {
		message = string([]rune(message)[:300])
	}
	_, err := s.DB.ExecContext(ctx, "UPDATE mail_outbox SET status=IF(attempts>=8,'FAILED','RETRY'),last_error=?,next_attempt_at=DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 60 SECOND),lease_until=NULL WHERE event_id=? AND attempts=? AND status='SENDING'", message, e.EventID, e.Attempts)
	return err
}

package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/notify"
)

type SupportInbox struct {
	notify.InboxSettings
	PasswordCipher, SourceHash, LeaseID string `json:"-"`
	UIDValidity, LastUID                uint32 `json:"-"`
}

func (s *Store) SupportInbox(ctx context.Context) (out SupportInbox, err error) {
	var raw string
	err = s.DB.QueryRowContext(ctx, "SELECT settings_json,password_cipher,version,source_hash,uid_validity,last_uid,last_poll_at,last_error,COALESCE(lease_id,'') FROM support_inbox WHERE id=1").Scan(&raw, &out.PasswordCipher, &out.Version, &out.SourceHash, &out.UIDValidity, &out.LastUID, &out.LastPollAt, &out.LastError, &out.LeaseID)
	if err != nil {
		return
	}
	version, last, lastError := out.Version, out.LastPollAt, out.LastError
	err = json.Unmarshal([]byte(raw), &out.InboxSettings)
	out.Version = version
	out.LastPollAt = last
	out.LastError = lastError
	out.PasswordConfigured = out.PasswordCipher != ""
	out.Initialized = out.UIDValidity != 0
	if out.Folder == "" {
		out.Folder = "INBOX"
	}
	if out.Port == 0 {
		out.Port = 993
	}
	if out.Host == "" {
		out.Host = "imap.qiye.aliyun.com"
	}
	if out.Address == "" {
		out.Address = "support@deuteriumix.com"
	}
	if out.Username == "" {
		out.Username = "support@deuteriumix.com"
	}
	return
}
func (s *Store) SaveSupportInbox(ctx context.Context, actor, request string, expected int64, settings notify.InboxSettings, password *string, key []byte) (json.RawMessage, error) {
	if settings.Validate() != nil || expected < 0 || password != nil && (len(*password) > 2048 || strings.ContainsRune(*password, 0)) {
		return nil, ErrSocialInvalid
	}
	settings.LastError = ""
	settings.LastPollAt = nil
	return s.socialMutate(ctx, actor, "support.inbox", request, struct {
		Version  int64
		Settings notify.InboxSettings
		Password *string
	}{expected, settings, password}, func(tx *sql.Tx) (any, error) {
		if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
			return nil, err
		}
		var version int64
		var previousValidity uint32
		var cipher, previous, previousJSON string
		if err := tx.QueryRowContext(ctx, "SELECT version,password_cipher,source_hash,settings_json,uid_validity FROM support_inbox WHERE id=1 FOR UPDATE").Scan(&version, &cipher, &previous, &previousJSON, &previousValidity); err != nil {
			return nil, err
		}
		if version != expected {
			return nil, ErrSocialVersion
		}
		if password != nil && *password != "" {
			var err error
			cipher, err = notify.Seal(key, *password)
			if err != nil {
				return nil, err
			}
		}
		if cipher == "" && settings.Enabled {
			return nil, ErrSocialInvalid
		}
		source := Digest([]byte(strings.ToLower(settings.Host) + ":" + strconv.Itoa(settings.Port) + ":" + settings.Username + ":" + settings.Folder + ":" + settings.Address))
		var oldSettings notify.InboxSettings
		if err := json.Unmarshal([]byte(previousJSON), &oldSettings); err != nil {
			return nil, err
		}
		settings.AcceptAfter = oldSettings.AcceptAfter
		if previous != source || (previousValidity == 0 && (settings.ImportExisting != oldSettings.ImportExisting || (!oldSettings.Enabled && settings.Enabled))) {
			settings.AcceptAfter = nil
			if !settings.ImportExisting {
				now := time.Now().UTC()
				settings.AcceptAfter = &now
			}
		}
		settings.Version = version + 1
		settings.Initialized = previousValidity != 0 && previous == source
		settings.PasswordConfigured = cipher != ""
		raw, _ := json.Marshal(settings)
		_, err := tx.ExecContext(ctx, "UPDATE support_inbox SET settings_json=?,password_cipher=?,version=?,source_hash=?,last_error='',lease_id=NULL,lease_until=NULL WHERE id=1", raw, cipher, settings.Version, source)
		// Use an explicit reset after saving; SQL assignment ordering must not decide whether the source changed.
		if err == nil && source != previous {
			_, err = tx.ExecContext(ctx, "UPDATE support_inbox SET uid_validity=0,last_uid=0 WHERE id=1")
		}
		if err != nil {
			return nil, err
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,'support.inbox','mailbox',UTC_TIMESTAMP(6))", actor)
		return settings, err
	})
}
func (s *Store) ClaimSupportInbox(ctx context.Context) (out SupportInbox, err error) {
	lease := ID("inbox_")
	r, err := s.DB.ExecContext(ctx, "UPDATE support_inbox SET lease_id=?,lease_until=DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 120 SECOND) WHERE id=1 AND JSON_UNQUOTE(JSON_EXTRACT(settings_json,'$.enabled'))='true' AND (lease_until IS NULL OR lease_until<UTC_TIMESTAMP(6))", lease)
	if err != nil {
		return
	}
	n, err := r.RowsAffected()
	if err != nil {
		return out, err
	}
	if n != 1 {
		return out, sql.ErrNoRows
	}
	out, err = s.SupportInbox(ctx)
	if err == nil && out.LeaseID != lease {
		err = sql.ErrNoRows
	}
	return
}
func (s *Store) SupportInboxPosition(ctx context.Context, inbox SupportInbox, validity, next uint32) (uint32, error) {
	if validity == 0 {
		return 0, errors.New("邮件服务器没有返回有效的 UIDVALIDITY。")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var old, after uint32
	if err = tx.QueryRowContext(ctx, "SELECT uid_validity,last_uid FROM support_inbox WHERE id=1 AND lease_id=? AND source_hash=? FOR UPDATE", inbox.LeaseID, inbox.SourceHash).Scan(&old, &after); err != nil {
		return 0, err
	}
	if old != validity {
		after = 0

		if _, err = tx.ExecContext(ctx, "UPDATE support_inbox SET uid_validity=?,last_uid=? WHERE id=1 AND lease_id=?", validity, after, inbox.LeaseID); err != nil {
			return 0, err
		}
	}
	return after, tx.Commit()
}
func (s *Store) FinishSupportInbox(ctx context.Context, lease string, pollErr error) error {
	message := ""
	if pollErr != nil {
		message = pollErr.Error()
		if len([]rune(message)) > 250 {
			message = string([]rune(message)[:250])
		}
	}
	_, err := s.DB.ExecContext(ctx, "UPDATE support_inbox SET last_poll_at=UTC_TIMESTAMP(6),last_error=?,lease_id=NULL,lease_until=NULL WHERE id=1 AND lease_id=?", message, lease)
	return err
}
func (s *Store) SupportInboxEvents(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT mail_uid,status,reason,COALESCE(ticket_id,''),received_at FROM support_inbound ORDER BY received_at DESC LIMIT 30")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var uid uint32
		var status, reason, ticket string
		var at sql.NullTime
		if err = rows.Scan(&uid, &status, &reason, &ticket, &at); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"uid": uid, "status": status, "reason": reason, "ticketId": ticket, "receivedAt": at.Time})
	}
	return out, rows.Err()
}

func enqueueThreadMailTx(ctx context.Context, tx *sql.Tx, channel, kind, recipient, subject, body, reference, actor, replyTo, inReplyTo string, headers map[string]string, key []byte) (string, error) {
	id, err := enqueueMailTx(ctx, tx, channel, kind, recipient, subject, body, reference, actor, key)
	if err != nil {
		return "", err
	}
	cipher := ""
	if len(headers) > 0 {
		raw, _ := json.Marshal(headers)
		cipher, err = notify.Seal(key, string(raw))
		if err != nil {
			return "", err
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE mail_outbox SET reply_to=?,in_reply_to=?,headers_cipher=? WHERE event_id=?", replyTo, inReplyTo, cipher, id); err != nil {
		return "", err
	}
	if kind == "TICKET_CREATED" || kind == "TICKET_REPLY" {
		messageID := id + "@deuterium-notifications"
		if _, err = tx.ExecContext(ctx, "INSERT INTO support_mail_links VALUES(?,?,?,?, 'OUTBOUND')", Digest([]byte(messageID)), messageID, reference, replyTo); err != nil {
			return "", err
		}
	}
	return id, nil
}

package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/notify"
	"strings"
	"time"
)

type SupportTicket struct {
	RequesterEmail string    `json:"requesterEmail"`
	Source         string    `json:"source"`
	ID             string    `json:"ticketId"`
	RequesterID    string    `json:"requesterId"`
	Subject        string    `json:"subject"`
	Category       string    `json:"category"`
	Status         string    `json:"status"`
	Priority       string    `json:"priority"`
	AssigneeID     string    `json:"assigneeId"`
	Version        int64     `json:"version"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	RequesterName  string    `json:"requesterName"`
}
type SupportMessage struct {
	AuthorEmail   string              `json:"authorEmail"`
	Direction     string              `json:"direction"`
	MailMessageID string              `json:"mailMessageId"`
	Attachments   []SupportAttachment `json:"attachments"`
	ID            string              `json:"messageId"`
	ActorID       string              `json:"actorId"`
	ActorName     string              `json:"actorName"`
	Visibility    string              `json:"visibility"`
	Body          string              `json:"body"`
	CreatedAt     time.Time           `json:"createdAt"`
}
type SupportSubmission struct {
	ClientRequestID string `json:"clientRequestId"`
	Subject         string `json:"subject"`
	Category        string `json:"category"`
	Body            string `json:"body"`
}
type SupportChange struct {
	Category        string  `json:"category"`
	ClientRequestID string  `json:"clientRequestId"`
	ExpectedVersion int64   `json:"expectedVersion"`
	Body            string  `json:"body"`
	Visibility      string  `json:"visibility"`
	Status          string  `json:"status"`
	Priority        string  `json:"priority"`
	AssigneeID      *string `json:"assigneeId"`
}

const supportSelect = `SELECT t.ticket_id,COALESCE(t.requester_id,''),t.subject,t.category,t.status,t.priority,COALESCE(t.assignee_id,''),t.version,t.created_at,t.updated_at,COALESCE(NULLIF(t.requester_name,''),t.requester_email),t.requester_email,t.source FROM support_tickets t LEFT JOIN identities i ON i.id=t.requester_id`

func scanSupport(row catalogScanner) (t SupportTicket, err error) {
	err = row.Scan(&t.ID, &t.RequesterID, &t.Subject, &t.Category, &t.Status, &t.Priority, &t.AssigneeID, &t.Version, &t.CreatedAt, &t.UpdatedAt, &t.RequesterName, &t.RequesterEmail, &t.Source)
	return
}
func requireSupportTx(ctx context.Context, tx *sql.Tx, actor string) error {
	var permitted bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM identities i JOIN identity_permissions p ON p.user_id=i.id WHERE i.id=? AND i.status='active' AND p.permission IN ('support.manage','platform.admin'))`, actor).Scan(&permitted); err != nil {
		return err
	}
	if !permitted {
		return ErrSocialNotFound
	}
	return nil
}
func (s *Store) SupportTickets(ctx context.Context, actor string, status, queue, search, category string, offset int) ([]SupportTicket, error) {
	if !catalogEnum(category, "", "ACCOUNT", "GAME", "ORDER", "ACCESS", "OTHER") || !catalogEnum(status, "", "OPEN", "IN_PROGRESS", "WAITING_CUSTOMER", "RESOLVED", "CLOSED") || !catalogEnum(queue, "", "mine", "unassigned") || !ValidSocialText(search, 0, 120) || offset < 0 {
		return nil, ErrSocialInvalid
	}
	where := " WHERE 1=1"
	args := []any{}

	if category != "" {
		where += " AND t.category=?"
		args = append(args, category)
	}
	if status != "" {
		where += " AND t.status=?"
		args = append(args, status)
	}
	if queue == "mine" {
		where += " AND t.assignee_id=?"
		args = append(args, actor)
	}
	if queue == "unassigned" {
		where += " AND t.assignee_id IS NULL"
	}
	if search != "" {
		where += " AND (LOCATE(LOWER(?),LOWER(t.subject))>0 OR LOCATE(?,t.ticket_id)>0 OR LOCATE(LOWER(?),LOWER(t.requester_email))>0 OR LOCATE(LOWER(?),LOWER(i.game_id))>0)"
		args = append(args, search, search, search, search)
	}
	rows, err := s.DB.QueryContext(ctx, supportSelect+where+" ORDER BY t.updated_at DESC,t.ticket_id DESC LIMIT 51 OFFSET ?", append(args, offset)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SupportTicket{}
	for rows.Next() {
		t, err := scanSupport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

type SupportAttachment struct {
	ID          string `json:"attachmentId"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}

func (s *Store) SupportDetail(ctx context.Context, id string) (map[string]any, error) {

	ticket, err := scanSupport(s.DB.QueryRowContext(ctx, supportSelect+" WHERE t.ticket_id=?", id))
	if err != nil {
		return nil, socialMissing(err)
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT m.message_id,COALESCE(m.actor_id,''),COALESCE(NULLIF(m.author_name,''),i.display_name,i.game_id,m.author_email),m.author_email,m.direction,m.mail_message_id,m.visibility,m.body,m.created_at FROM support_messages m LEFT JOIN identities i ON i.id=m.actor_id WHERE m.ticket_id=? ORDER BY m.created_at,m.message_id`, id)
	if err != nil {
		return nil, err
	}
	messages := []SupportMessage{}
	for rows.Next() {
		var m SupportMessage
		if err = rows.Scan(&m.ID, &m.ActorID, &m.ActorName, &m.AuthorEmail, &m.Direction, &m.MailMessageID, &m.Visibility, &m.Body, &m.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		m.Attachments = []SupportAttachment{}
		messages = append(messages, m)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	attachments, err := s.DB.QueryContext(ctx, "SELECT attachment_id,message_id,filename,mime_type,size_bytes FROM support_attachments WHERE ticket_id=?", id)
	if err != nil {
		return nil, err
	}
	for attachments.Next() {
		var a SupportAttachment
		var messageID string
		if err = attachments.Scan(&a.ID, &messageID, &a.Filename, &a.ContentType, &a.Size); err != nil {
			attachments.Close()
			return nil, err
		}
		for n := range messages {
			if messages[n].ID == messageID {
				messages[n].Attachments = append(messages[n].Attachments, a)
			}
		}
	}
	err = attachments.Err()
	attachments.Close()
	if err != nil {
		return nil, err
	}
	requester := map[string]any{"email": ticket.RequesterEmail, "gameId": "", "qq": "", "gameVerified": false, "status": "email", "source": ticket.Source}
	if ticket.RequesterID != "" {
		var qq, game, status, ref string
		var verified bool
		err = s.DB.QueryRowContext(ctx, "SELECT qq,game_id,status,player_ref,game_verified FROM identities WHERE id=?", ticket.RequesterID).Scan(&qq, &game, &status, &ref, &verified)
		if err != nil {
			return nil, err
		}
		requester["qq"] = qq
		requester["gameId"] = game
		requester["status"] = status
		requester["playerRef"] = ref
		requester["gameVerified"] = verified
	}
	var count int
	if err = s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM support_tickets WHERE requester_email=?", ticket.RequesterEmail).Scan(&count); err != nil {
		return nil, err
	}
	requester["ticketCount"] = count
	rows, err = s.DB.QueryContext(ctx, "SELECT event_id,status,last_error,kind FROM mail_outbox WHERE reference_id=? ORDER BY created_at DESC LIMIT 20", id)
	if err != nil {
		return nil, err
	}
	delivery := []map[string]string{}
	for rows.Next() {
		var event, state, message, kind string
		if err = rows.Scan(&event, &state, &message, &kind); err != nil {
			rows.Close()
			return nil, err
		}
		delivery = append(delivery, map[string]string{"eventId": event, "status": state, "lastError": message, "kind": kind})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	return map[string]any{"ticket": ticket, "messages": messages, "requester": requester, "delivery": delivery}, nil
}
func (s *Store) SupportAttachment(ctx context.Context, ticket, id string, key []byte) (SupportAttachment, []byte, error) {
	var a SupportAttachment
	var cipher string
	err := s.DB.QueryRowContext(ctx, "SELECT attachment_id,filename,mime_type,size_bytes,content_cipher FROM support_attachments WHERE attachment_id=? AND ticket_id=?", id, ticket).Scan(&a.ID, &a.Filename, &a.ContentType, &a.Size, &cipher)
	if err != nil {
		return a, nil, socialMissing(err)
	}
	plain, err := notify.Open(key, cipher)
	return a, []byte(plain), err
}
func (s *Store) ChangeSupport(ctx context.Context, actor, id string, in SupportChange, key []byte) (json.RawMessage, error) {

	in.Body = strings.TrimSpace(in.Body)
	if in.ExpectedVersion < 1 || !catalogEnum(in.Category, "", "ACCOUNT", "GAME", "ORDER", "ACCESS", "OTHER") || !ValidSocialText(in.Body, 0, 12000) || !catalogEnum(in.Visibility, "", "PUBLIC", "INTERNAL") || !catalogEnum(in.Status, "", "OPEN", "IN_PROGRESS", "WAITING_CUSTOMER", "RESOLVED", "CLOSED") || !catalogEnum(in.Priority, "", "LOW", "NORMAL", "HIGH", "URGENT") {
		return nil, ErrSocialInvalid
	}

	if in.Body == "" && in.Status == "" && in.Priority == "" && in.AssigneeID == nil && in.Category == "" {
		return nil, ErrSocialInvalid
	}
	return s.socialMutate(ctx, actor, "support.change:"+id, in.ClientRequestID, in, func(tx *sql.Tx) (any, error) {
		if err := requireSupportTx(ctx, tx, actor); err != nil {
			return nil, err
		}
		var owner, status, subject, email string
		var version int64
		if err := tx.QueryRowContext(ctx, "SELECT COALESCE(requester_id,''),status,subject,version FROM support_tickets WHERE ticket_id=? FOR UPDATE", id).Scan(&owner, &status, &subject, &version); err != nil {
			return nil, socialMissing(err)
		}

		if version != in.ExpectedVersion {
			return nil, ErrSocialVersion
		}
		if in.AssigneeID != nil && *in.AssigneeID != "" {
			if err := requireSupportTx(ctx, tx, *in.AssigneeID); err != nil {
				return nil, err
			}
		}
		if in.Body != "" {
			visibility := in.Visibility
			if visibility == "" {
				visibility = "PUBLIC"
			}
			direction := "OUTBOUND"
			if visibility == "INTERNAL" {
				direction = "NOTE"
			}
			var actorName string
			if err := tx.QueryRowContext(ctx, "SELECT COALESCE(display_name,game_id) FROM identities WHERE id=?", actor).Scan(&actorName); err != nil {
				return nil, err
			}
			replyID := ID("reply_")
			if _, err := tx.ExecContext(ctx, "INSERT INTO support_messages(message_id,ticket_id,actor_id,author_name,direction,visibility,body,created_at) VALUES(?,?,?,?,?,?,?,UTC_TIMESTAMP(6))", replyID, id, actor, actorName, direction, visibility, in.Body); err != nil {
				return nil, err
			}
			if visibility == "PUBLIC" {
				if err := tx.QueryRowContext(ctx, "SELECT requester_email FROM support_tickets WHERE ticket_id=?", id).Scan(&email); err != nil {
					return nil, err
				}
				var inboxJSON string
				if err := tx.QueryRowContext(ctx, "SELECT settings_json FROM support_inbox WHERE id=1").Scan(&inboxJSON); err != nil {
					return nil, err
				}
				var inbox notify.InboxSettings
				if json.Unmarshal([]byte(inboxJSON), &inbox) != nil || inbox.Address == "" {
					return nil, catalogError(409, "SUPPORT_INBOX_NOT_READY", "请先配置客服邮箱。")
				}
				var latest string
				if err := tx.QueryRowContext(ctx, "SELECT mail_message_id FROM support_messages WHERE ticket_id=? AND direction='INBOUND' ORDER BY created_at DESC,message_id DESC LIMIT 1", id).Scan(&latest); err != nil {
					return nil, err
				}
				event, err := enqueueThreadMailTx(ctx, tx, "ENTERPRISE", "TICKET_REPLY", email, "[工单 "+id+"] "+subject, in.Body, id, actor, inbox.Address, latest, nil, key)
				if err != nil {
					return nil, err
				}
				if _, err = tx.ExecContext(ctx, "UPDATE support_messages SET author_email=?,mail_message_id=? WHERE message_id=?", inbox.Address, event+"@deuterium-notifications", replyID); err != nil {
					return nil, err
				}
			}
			if visibility == "PUBLIC" {
				status = "WAITING_CUSTOMER"
			}
		}
		if in.Status != "" {
			status = in.Status
		}
		if _, err := tx.ExecContext(ctx, "UPDATE support_tickets SET status=?,version=version+1,updated_at=UTC_TIMESTAMP(6) WHERE ticket_id=?", status, id); err != nil {
			return nil, err
		}
		if in.Category != "" {
			if _, err := tx.ExecContext(ctx, "UPDATE support_tickets SET category=? WHERE ticket_id=?", in.Category, id); err != nil {
				return nil, err
			}
		}
		if in.Priority != "" {
			if _, err := tx.ExecContext(ctx, "UPDATE support_tickets SET priority=? WHERE ticket_id=?", in.Priority, id); err != nil {
				return nil, err
			}
		}
		if in.AssigneeID != nil {
			if _, err := tx.ExecContext(ctx, "UPDATE support_tickets SET assignee_id=? WHERE ticket_id=?", nullString(*in.AssigneeID), id); err != nil {
				return nil, err
			}
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,'support.update',?,UTC_TIMESTAMP(6))", actor, id); err != nil {
			return nil, err
		}
		return map[string]any{"ticketId": id, "version": version + 1, "status": status}, nil
	})
}

type SupportMacro struct {
	ID      string `json:"macroId"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	Version int64  `json:"version"`
}

func (s *Store) SupportTools(ctx context.Context) (map[string]any, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT macro_id,title,body,version FROM support_macros ORDER BY title")
	if err != nil {
		return nil, err
	}
	macros := []SupportMacro{}
	for rows.Next() {
		var m SupportMacro
		if err = rows.Scan(&m.ID, &m.Title, &m.Body, &m.Version); err != nil {
			rows.Close()
			return nil, err
		}
		macros = append(macros, m)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = s.DB.QueryContext(ctx, `SELECT i.id,COALESCE(i.display_name,NULLIF(i.game_id,''),i.email,'管理员') FROM identities i WHERE i.status='active' AND EXISTS(SELECT 1 FROM identity_permissions p WHERE p.user_id=i.id AND p.permission IN ('support.manage','platform.admin')) ORDER BY i.game_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	agents := []map[string]string{}
	for rows.Next() {
		var id, name string
		if err = rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		agents = append(agents, map[string]string{"userId": id, "name": name})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	counts := map[string]int{}
	rows, err = s.DB.QueryContext(ctx, "SELECT status,COUNT(*) FROM support_tickets GROUP BY status")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var state string
		var count int
		if err = rows.Scan(&state, &count); err != nil {
			rows.Close()
			return nil, err
		}
		counts[state] = count
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	inbox, err := s.SupportInbox(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"macros": macros, "agents": agents, "counts": counts, "mailbox": map[string]any{"email": inbox.Address, "enabled": inbox.Enabled, "lastPollAt": inbox.LastPollAt, "lastError": inbox.LastError}}, nil
}
func (s *Store) SaveSupportMacro(ctx context.Context, actor, key string, m SupportMacro) (json.RawMessage, error) {
	if !ValidSocialID(m.ID) || !ValidSocialText(m.Title, 1, 100) || !ValidSocialText(m.Body, 1, 6000) || m.Version < 0 {
		return nil, ErrSocialInvalid
	}
	return s.socialMutate(ctx, actor, "support.macro:"+m.ID, key, m, func(tx *sql.Tx) (any, error) {
		if err := requireSupportTx(ctx, tx, actor); err != nil {
			return nil, err
		}
		if m.Version == 0 {
			if _, err := tx.ExecContext(ctx, "INSERT INTO support_macros VALUES(?,?,?,1,UTC_TIMESTAMP(6))", m.ID, m.Title, m.Body); err != nil {
				if duplicate(err) {
					return nil, ErrSocialVersion
				}
				return nil, err
			}
		} else {
			r, err := tx.ExecContext(ctx, "UPDATE support_macros SET title=?,body=?,version=version+1,updated_at=UTC_TIMESTAMP(6) WHERE macro_id=? AND version=?", m.Title, m.Body, m.ID, m.Version)
			if err != nil {
				return nil, err
			}
			n, err := r.RowsAffected()
			if err != nil {
				return nil, err
			}
			if n != 1 {
				return nil, ErrSocialVersion
			}
		}
		m.Version++
		return m, nil
	})
}

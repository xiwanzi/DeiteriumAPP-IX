package store

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/notify"
)

var supportSubjectNumber = regexp.MustCompile(`\[工单\s*([0-9]{12})\]`)

func ticketNumberTx(ctx context.Context, tx *sql.Tx, now time.Time) (string, error) {
	day := now.In(time.FixedZone("UTC+8", 8*3600)).Format("20060102")
	if _, err := tx.ExecContext(ctx, "INSERT IGNORE INTO support_ticket_days VALUES(?)", day); err != nil {
		return "", err
	}
	var lock string
	if err := tx.QueryRowContext(ctx, "SELECT day_key FROM support_ticket_days WHERE day_key=? FOR UPDATE", day).Scan(&lock); err != nil {
		return "", err
	}
	rows, err := tx.QueryContext(ctx, "SELECT ticket_id FROM support_tickets WHERE ticket_id LIKE ?", day+"%")
	if err != nil {
		return "", err
	}
	used := map[string]bool{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return "", err
		}
		used[id] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return "", err
	}
	if len(used) >= 10000 {
		return "", errors.New("当日四位工单编号已用完，来信保留待后续处理。")
	}
	// Uniformly choose from unused four-digit values under the daily lock.
	// Concurrent receipt cannot collide, including values with leading zeroes.
	pick, err := rand.Int(rand.Reader, big.NewInt(int64(10000-len(used))))
	if err != nil {
		return "", err
	}
	n := int(pick.Int64())
	for suffix := 0; suffix < 10000; suffix++ {
		id := fmt.Sprintf("%s%04d", day, suffix)
		if used[id] {
			continue
		}
		if n == 0 {
			return id, nil
		}
		n--
	}
	return "", ErrConflict
}
func guessSupportCategory(subject string) string {
	for _, v := range []struct{ key, words string }{{"ACCESS", "白名单|申请|审核|通行"}, {"ORDER", "订单|退款|购买|付款|商品"}, {"ACCOUNT", "账号|验证|密码|登录|邮箱|绑定"}, {"GAME", "游戏|服务器|掉线|背包|建筑|插件"}} {
		for _, word := range strings.Split(v.words, "|") {
			if strings.Contains(subject, word) {
				return v.key
			}
		}
	}
	return "OTHER"
}

// Only the receiver creates a ticket. Help submissions are ordinary outbound
// system mail carrying a one-time server-bound proof and the player's Reply-To.
func (s *Store) SubmitSupportMail(ctx context.Context, user string, in SupportSubmission, key []byte) (json.RawMessage, error) {
	in.Subject = strings.TrimSpace(in.Subject)
	in.Body = strings.TrimSpace(in.Body)
	if !ValidSocialText(in.Subject, 2, 120) || strings.ContainsAny(in.Subject, "\r\n") || !ValidSocialText(in.Body, 2, 12000) || !catalogEnum(in.Category, "ACCOUNT", "GAME", "ORDER", "ACCESS", "OTHER") {
		return nil, ErrSocialInvalid
	}
	return s.socialMutate(ctx, user, "support.submit-mail", in.ClientRequestID, in, func(tx *sql.Tx) (any, error) {
		var email, name string
		if err := tx.QueryRowContext(ctx, "SELECT COALESCE(email,CONCAT(qq,'@qq.com')),COALESCE(display_name,game_id) FROM identities WHERE id=? FOR UPDATE", user).Scan(&email, &name); err != nil {
			return nil, err
		}
		if _, err := NormalizeEmail(email); err != nil {
			return nil, ErrSocialInvalid
		}
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM support_submissions WHERE user_id=? AND created_at>DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 DAY)", user).Scan(&count); err != nil {
			return nil, err
		}
		if count >= 10 {
			return nil, ErrRateLimited
		}
		var inboxJSON, systemJSON string
		if err := tx.QueryRowContext(ctx, "SELECT settings_json FROM support_inbox WHERE id=1").Scan(&inboxJSON); err != nil {
			return nil, err
		}
		if err := tx.QueryRowContext(ctx, "SELECT settings_json FROM mail_channels WHERE channel_id='SYSTEM'").Scan(&systemJSON); err != nil {
			return nil, catalogError(503, "HELP_MAIL_UNAVAILABLE", "邮件提交服务尚未配置。")
		}
		var inbox notify.InboxSettings
		var system notify.Settings
		if json.Unmarshal([]byte(inboxJSON), &inbox) != nil || json.Unmarshal([]byte(systemJSON), &system) != nil || !inbox.Enabled || !system.Enabled || inbox.Address == system.From {
			return nil, catalogError(503, "HELP_MAIL_UNAVAILABLE", "邮件提交服务暂不可用。")
		}
		proofBytes := make([]byte, 32)
		if _, err := rand.Read(proofBytes); err != nil {
			return nil, err
		}
		proof := hex.EncodeToString(proofBytes)
		id := ID("help_")
		if _, err := tx.ExecContext(ctx, "INSERT INTO support_submissions(submission_id,user_id,sender_email,email,name,category,subject,body,proof_hash,created_at) VALUES(?,?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6))", id, user, system.From, email, name, in.Category, in.Subject, in.Body, Digest([]byte(proof))); err != nil {
			return nil, err
		}
		body := fmt.Sprintf("通过 Deuterium Help 提交\n玩家：%s\n邮箱：%s\n问题类型：%s\n\n%s", name, email, in.Category, in.Body)
		event, err := enqueueThreadMailTx(ctx, tx, "SYSTEM", "HELP_SUBMISSION", inbox.Address, in.Subject, body, id, user, email, "", map[string]string{"X-Deuterium-Submission": id, "X-Deuterium-Proof": proof}, key)
		return map[string]any{"submissionId": id, "eventId": event, "status": "QUEUED"}, err
	})
}

type IngestResult struct {
	TicketID string `json:"ticketId"`
	Status   string `json:"status"`
}

func (s *Store) IngestSupportMail(ctx context.Context, inbox SupportInbox, validity, uid uint32, raw []byte, receivedAt time.Time, key []byte) (result IngestResult, err error) {
	message, parseErr := notify.ReadInbound(raw)
	tx, err := s.beginAccountTx(ctx, "")
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	var position uint32
	if err = tx.QueryRowContext(ctx, "SELECT last_uid FROM support_inbox WHERE id=1 AND lease_id=? AND source_hash=? AND uid_validity=? FOR UPDATE", inbox.LeaseID, inbox.SourceHash, validity).Scan(&position); err != nil {
		return result, err
	}
	var seen int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM support_inbound WHERE source_hash=? AND uid_validity=? AND mail_uid=?", inbox.SourceHash, validity, uid).Scan(&seen); err != nil {
		return result, err
	}
	if seen > 0 {
		return IngestResult{Status: "DUPLICATE"}, nil
	}
	finish := func(status, reason, ticket string) (IngestResult, error) {
		if len([]rune(reason)) > 250 {
			reason = string([]rune(reason)[:250])
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO support_inbound VALUES(?,?,?,?,?,?,?,UTC_TIMESTAMP(6))", inbox.SourceHash, validity, uid, Digest(raw), nullString(ticket), status, reason); err != nil {
			return IngestResult{}, err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE support_inbox SET last_uid=GREATEST(last_uid,?) WHERE id=1 AND lease_id=?", uid, inbox.LeaseID); err != nil {
			return IngestResult{}, err
		}
		return IngestResult{TicketID: ticket, Status: status}, tx.Commit()
	}
	if inbox.AcceptAfter != nil && !receivedAt.IsZero() && receivedAt.Before(*inbox.AcceptAfter) {
		return finish("IGNORED", "接入前的历史邮件保留在邮箱，不自动建单。", "")
	}
	if parseErr != nil {
		return finish("NEEDS_REVIEW", parseErr.Error(), "")
	}
	customer, err := NormalizeEmail(message.ReplyTo)
	if err != nil {
		return finish("NEEDS_REVIEW", "回复邮箱格式无效，请在邮箱中处理。", "")
	}
	sender, err := NormalizeEmail(message.From)
	if err != nil {
		return finish("NEEDS_REVIEW", "发件邮箱格式无效。", "")
	}
	if message.MessageID == "" {
		message.MessageID = Digest(raw) + "@deuterium-inbound"
	}
	fingerprint := Digest([]byte(message.MessageID + ":" + sender))
	var duplicateTicket string
	err = tx.QueryRowContext(ctx, "SELECT ticket_id FROM support_mail_links WHERE message_hash=?", fingerprint).Scan(&duplicateTicket)
	if err == nil {
		return finish("DUPLICATE", "同一封邮件已处理。", duplicateTicket)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return result, err
	}
	source, category, owner, name := "EMAIL", guessSupportCategory(message.Subject), "", message.FromName
	var submissionID string
	if message.SubmissionID != "" {
		var storedEmail, storedSender, storedProof, existing string
		err = tx.QueryRowContext(ctx, "SELECT user_id,sender_email,email,name,category,subject,body,proof_hash,COALESCE(ticket_id,'') FROM support_submissions WHERE submission_id=? FOR UPDATE", message.SubmissionID).Scan(&owner, &storedSender, &storedEmail, &name, &category, &message.Subject, &message.Text, &storedProof, &existing)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return result, err
		}
		if err != nil || strings.ToLower(storedSender) != sender || storedEmail != customer || subtle.ConstantTimeCompare([]byte(storedProof), []byte(Digest([]byte(message.Proof)))) != 1 {
			return finish("NEEDS_REVIEW", "系统转交邮件缺少有效的玩家绑定证明。", "")
		}
		if existing != "" {
			return finish("DUPLICATE", "Help 邮件已建立工单。", existing)
		}
		source = "HELP"
		submissionID = message.SubmissionID
	} else {
		if message.Automated {
			return finish("IGNORED", "自动回复或退信，避免邮件循环。", "")
		}
		var self int
		if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM mail_channels WHERE LOWER(JSON_UNQUOTE(JSON_EXTRACT(settings_json,'$.from'))) IN (?,?)", sender, customer).Scan(&self); err != nil {
			return result, err
		}
		if self > 0 || sender == strings.ToLower(inbox.Address) || customer == strings.ToLower(inbox.Address) {
			return finish("IGNORED", "系统自身邮件不建立客户工单。", "")
		}
		err = tx.QueryRowContext(ctx, "SELECT id,COALESCE(display_name,game_id) FROM identities WHERE status='active' AND (email=? OR (email IS NULL AND CONCAT(qq,'@qq.com')=?)) ORDER BY id LIMIT 1", customer, customer).Scan(&owner, &name)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return result, err
		}
	}
	ticket := ""
	if source != "HELP" {
		for n := len(message.References) - 1; n >= 0; n-- {
			err = tx.QueryRowContext(ctx, "SELECT t.ticket_id FROM support_mail_links l JOIN support_tickets t ON t.ticket_id=l.ticket_id WHERE l.message_id=? AND t.requester_email=? ORDER BY t.created_at DESC LIMIT 1", message.References[n], customer).Scan(&ticket)
			if err == nil {
				break
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return result, err
			}
		}
	}
	if ticket == "" && source != "HELP" {
		if match := supportSubjectNumber.FindStringSubmatch(message.Subject); len(match) == 2 {
			err = tx.QueryRowContext(ctx, "SELECT ticket_id FROM support_tickets WHERE ticket_id=? AND requester_email=?", match[1], customer).Scan(&ticket)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return result, err
			}
		}
	}
	created := ticket == ""
	if created {
		ticket, err = ticketNumberTx(ctx, tx, time.Now().UTC())
		if err != nil {
			return result, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO support_tickets(ticket_id,requester_id,requester_email,requester_name,source,subject,category,created_at,updated_at) VALUES(?,?,?,?,?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))", ticket, nullString(owner), customer, name, source, message.Subject, category); err != nil {
			return result, err
		}
	} else {
		if _, err = tx.ExecContext(ctx, "UPDATE support_tickets SET status='OPEN',version=version+1,updated_at=UTC_TIMESTAMP(6) WHERE ticket_id=?", ticket); err != nil {
			return result, err
		}
	}
	messageID := ID("reply_")
	if _, err = tx.ExecContext(ctx, "INSERT INTO support_messages(message_id,ticket_id,actor_id,author_name,author_email,direction,mail_message_id,visibility,body,created_at) VALUES(?,?,?,?,?,'INBOUND',?,'PUBLIC',?,UTC_TIMESTAMP(6))", messageID, ticket, nullString(owner), name, sender, message.MessageID, message.Text); err != nil {
		return result, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO support_mail_links VALUES(?,?,?,?, 'INBOUND')", fingerprint, message.MessageID, ticket, sender); err != nil {
		return result, err
	}
	for _, attachment := range message.Attachments {
		cipher, e := notify.Seal(key, string(attachment.Data))
		if e != nil {
			return result, e
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO support_attachments VALUES(?,?,?,?,?,?,?)", ID("attach_"), ticket, messageID, attachment.Name, attachment.ContentType, len(attachment.Data), cipher); err != nil {
			return result, err
		}
	}
	if submissionID != "" {
		if _, err = tx.ExecContext(ctx, "UPDATE support_submissions SET ticket_id=? WHERE submission_id=?", ticket, submissionID); err != nil {
			return result, err
		}
	}
	if created {
		if _, err = enqueueThreadMailTx(ctx, tx, "DM", "TICKET_CREATED", customer, message.Subject, message.Text, ticket, "", inbox.Address, message.MessageID, nil, key); err != nil {
			return result, err
		}
	}
	return finish("IMPORTED", "", ticket)
}

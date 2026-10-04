package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/notify"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func (s *Server) registerMailSupport(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/admin/mail/templates/verification/{purpose}", func(w http.ResponseWriter, r *http.Request) {
		u, err := s.admin(r, "platform.admin")
		if err != nil {
			failError(w, r, err)
			return
		}
		m, err := notify.RenderVerification(notify.VerificationMail{TemplateVersion: notify.VerificationTemplateVersion, Purpose: r.PathValue("purpose"), GameID: u.GameID, Code: "482691"})
		if err != nil {
			failure(w, r, 400, "VERIFICATION_TEMPLATE_INVALID", err.Error())
			return
		}
		v2Success(w, r, map[string]any{"subject": m.Subject, "html": notify.AdmissionPreviewHTML(m), "preview": true, "purpose": r.PathValue("purpose")})
	})
	for _, path := range []string{"/api/v1/admin/mail/events/{eventId}/retry", "/api/v1/admin/support/tickets/{ticketId}/mail/{eventId}/retry"} {
		mux.HandleFunc("POST "+path, func(w http.ResponseWriter, r *http.Request) {
			permission := "platform.admin"
			if r.PathValue("ticketId") != "" {
				permission = "support.manage"
			}
			u, err := s.admin(r, permission)
			if err != nil {
				failError(w, r, err)
				return
			}
			var in struct {
				ClientRequestID string `json:"clientRequestId"`
			}
			if err = body(w, r, &in); err != nil {
				failError(w, r, err)
				return
			}
			out, err := s.Store.RetrySystemMail(r.Context(), u.ID, r.PathValue("eventId"), r.PathValue("ticketId"), in.ClientRequestID)
			if err != nil {
				catalogFailV2(w, r, err)
				return
			}
			v2Success(w, r, out)
		})
	}
	mux.HandleFunc("GET /api/v1/admin/mail", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		channels := map[string]store.EmailSettingsV204{}
		for _, id := range []string{"DM", "ENTERPRISE", "SYSTEM"} {
			v, err := s.Store.MailChannel(r.Context(), id)
			if err != nil {
				failError(w, r, err)
				return
			}
			channels[id] = v
		}
		events, err := s.Store.SystemMailEvents(r.Context())
		if err != nil {
			failError(w, r, err)
			return
		}
		notifications, err := s.Store.EmailSettingsV204(r.Context())
		if err != nil {
			failError(w, r, err)
			return
		}
		legacy, err := s.Store.EmailStatusV204(r.Context())
		if err != nil {
			failError(w, r, err)
			return
		}
		inbox, err := s.Store.SupportInbox(r.Context())
		if err != nil {
			failError(w, r, err)
			return
		}
		inbound, err := s.Store.SupportInboxEvents(r.Context())
		if err != nil {
			failError(w, r, err)
			return
		}
		v2Success(w, r, map[string]any{"inbox": inbox.InboxSettings, "inbound": inbound, "channels": channels, "events": events, "legacyDelivery": legacy, "notifications": map[string]any{"version": notifications.Version, "recipients": notifications.Recipients, "admissionReviewEnabled": notifications.AdmissionReviewEnabled}, "secretStorageReady": len(s.Config.SMTPKey) == 32})
	})
	mux.HandleFunc("PUT /api/v1/admin/mail/notifications", func(w http.ResponseWriter, r *http.Request) {
		u, err := s.admin(r, "platform.admin")
		if err != nil {
			failError(w, r, err)
			return
		}
		var in struct {
			ClientRequestID        string   `json:"clientRequestId"`
			ExpectedVersion        int64    `json:"expectedVersion"`
			AdmissionReviewEnabled bool     `json:"admissionReviewEnabled"`
			Recipients             []string `json:"recipients"`
		}
		if err = body(w, r, &in); err != nil {
			failError(w, r, err)
			return
		}
		out, err := s.Store.SaveMailNotifications(r.Context(), u.ID, in.ClientRequestID, in.ExpectedVersion, in.AdmissionReviewEnabled, in.Recipients)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, out)
	})
	mux.HandleFunc("PUT /api/v1/admin/mail/channels/{channel}", func(w http.ResponseWriter, r *http.Request) {
		u, err := s.admin(r, "platform.admin")
		if err != nil {
			failError(w, r, err)
			return
		}
		var in struct {
			ClientRequestID string          `json:"clientRequestId"`
			ExpectedVersion int64           `json:"expectedVersion"`
			Settings        notify.Settings `json:"settings"`
			Password        *string         `json:"password"`
		}
		if err = body(w, r, &in); err != nil {
			failError(w, r, err)
			return
		}
		result, err := s.Store.SaveMailChannel(r.Context(), u.ID, r.PathValue("channel"), in.ClientRequestID, in.ExpectedVersion, in.Settings, in.Password, s.Config.SMTPKey)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, result)
	})
	mux.HandleFunc("POST /api/v1/admin/mail/send", func(w http.ResponseWriter, r *http.Request) {
		u, err := s.admin(r, "platform.admin")
		if err != nil {
			failError(w, r, err)
			return
		}
		var in struct {
			ClientRequestID string `json:"clientRequestId"`
			Recipient       string `json:"recipient"`
			Subject         string `json:"subject"`
			Body            string `json:"body"`
			Channel         string `json:"channel"`
		}
		if err = socialBodyV2(w, r, &in, 64<<10); err != nil {
			failError(w, r, err)
			return
		}
		if in.Channel == "" {
			in.Channel = "ENTERPRISE"
		}
		result, err := s.Store.QueueManualMail(r.Context(), u.ID, in.ClientRequestID, in.Channel, in.Recipient, in.Subject, in.Body, s.Config.SMTPKey)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, result)
	})
	mux.HandleFunc("POST /api/v1/support/submissions", func(w http.ResponseWriter, r *http.Request) {
		u, err := s.authenticate(r)
		if err != nil {
			failError(w, r, err)
			return
		}
		var in store.SupportSubmission
		if err = socialBodyV2(w, r, &in, 64<<10); err != nil {
			failError(w, r, err)
			return
		}
		result, err := s.Store.SubmitSupportMail(r.Context(), u.User.ID, in, s.Config.SMTPKey)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, result)
	})
	{
		base := "/api/v1/admin/support/tickets"
		actor := func(r *http.Request) (store.User, error) { return s.admin(r, "support.manage") }
		mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) {
			u, err := actor(r)
			if err != nil {
				failError(w, r, err)
				return
			}
			offset := 0
			if raw := r.URL.Query().Get("offset"); raw != "" {
				offset, err = strconv.Atoi(raw)
				if err != nil {
					socialFailureV2(w, r, store.ErrSocialInvalid)
					return
				}
			}
			items, err := s.Store.SupportTickets(r.Context(), u.ID, r.URL.Query().Get("status"), r.URL.Query().Get("queue"), r.URL.Query().Get("q"), r.URL.Query().Get("category"), offset)
			if err != nil {
				catalogFailV2(w, r, err)
				return
			}
			more := len(items) > 50
			if more {
				items = items[:50]
			}
			v2Success(w, r, map[string]any{"items": items, "hasMore": more})
		})
		mux.HandleFunc("GET "+base+"/{ticketId}", func(w http.ResponseWriter, r *http.Request) {
			_, err := actor(r)
			if err != nil {
				failError(w, r, err)
				return
			}
			result, err := s.Store.SupportDetail(r.Context(), r.PathValue("ticketId"))
			if err != nil {
				catalogFailV2(w, r, err)
				return
			}
			v2Success(w, r, result)
		})
		mux.HandleFunc("POST "+base+"/{ticketId}", func(w http.ResponseWriter, r *http.Request) {
			u, err := actor(r)
			if err != nil {
				failError(w, r, err)
				return
			}
			var in store.SupportChange
			if err = socialBodyV2(w, r, &in, 64<<10); err != nil {
				failError(w, r, err)
				return
			}
			result, err := s.Store.ChangeSupport(r.Context(), u.ID, r.PathValue("ticketId"), in, s.Config.SMTPKey)
			if err != nil {
				catalogFailV2(w, r, err)
				return
			}
			v2Success(w, r, result)
		})
	}

	mux.HandleFunc("GET /api/v1/support/contact", func(w http.ResponseWriter, r *http.Request) {
		inbox, err := s.Store.SupportInbox(r.Context())
		if err != nil {
			failError(w, r, err)
			return
		}
		success(w, r, map[string]any{"email": inbox.Address, "enabled": inbox.Enabled && inbox.Address != ""})
	})
	mux.HandleFunc("PUT /api/v1/admin/mail/inbox", func(w http.ResponseWriter, r *http.Request) {
		u, err := s.admin(r, "platform.admin")
		if err != nil {
			failError(w, r, err)
			return
		}
		var in struct {
			ClientRequestID string               `json:"clientRequestId"`
			ExpectedVersion int64                `json:"expectedVersion"`
			Settings        notify.InboxSettings `json:"settings"`
			Password        *string              `json:"password"`
		}
		if err = body(w, r, &in); err != nil {
			failError(w, r, err)
			return
		}
		out, err := s.Store.SaveSupportInbox(r.Context(), u.ID, in.ClientRequestID, in.ExpectedVersion, in.Settings, in.Password, s.Config.SMTPKey)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, out)
	})
	mux.HandleFunc("GET /api/v1/admin/support/tickets/{ticketId}/attachments/{attachmentId}", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "support.manage"); err != nil {
			failError(w, r, err)
			return
		}
		a, data, err := s.Store.SupportAttachment(r.Context(), r.PathValue("ticketId"), r.PathValue("attachmentId"), s.Config.SMTPKey)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": a.Filename}))
		w.Write(data)
	})
	mux.HandleFunc("GET /api/v1/admin/mail/templates/support-received", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		m := notify.RenderSupportAcknowledgement(time.Now().In(time.FixedZone("UTC+8", 8*3600)).Format("20060102")+"0427", "游戏登录遇到问题", "玩家已提供问题类型和详细信息。支持人员会通过邮件继续联系。")
		success(w, r, map[string]string{"subject": m.Subject, "html": notify.AdmissionPreviewHTML(m)})
	})
	mux.HandleFunc("GET /api/v1/admin/support/tools", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "support.manage"); err != nil {
			failError(w, r, err)
			return
		}
		result, err := s.Store.SupportTools(r.Context())
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, result)
	})
	mux.HandleFunc("PUT /api/v1/admin/support/macros/{macroId}", func(w http.ResponseWriter, r *http.Request) {
		u, err := s.admin(r, "support.manage")
		if err != nil {
			failError(w, r, err)
			return
		}
		var in struct {
			ClientRequestID string `json:"clientRequestId"`
			store.SupportMacro
		}
		if err = socialBodyV2(w, r, &in, 32<<10); err != nil {
			failError(w, r, err)
			return
		}
		in.ID = r.PathValue("macroId")
		result, err := s.Store.SaveSupportMacro(r.Context(), u.ID, in.ClientRequestID, in.SupportMacro)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, result)
	})
}

func (s *Server) processSystemMail(ctx context.Context, send emailSenderV204) error {
	event, err := s.Store.ClaimMail(ctx)
	if errors.Is(err, store.ErrMailSkipped) {
		return nil
	}
	if err != nil {
		return err
	}
	channel, sendErr := s.Store.MailChannel(ctx, event.Channel)
	if sendErr == nil && !channel.Enabled {
		sendErr = sql.ErrNoRows
	}
	var password, text string
	if sendErr == nil {
		password, sendErr = notify.Open(s.Config.SMTPKey, channel.PasswordCipher)
	}
	if sendErr == nil {
		text, sendErr = notify.Open(s.Config.SMTPKey, event.Cipher)
	}
	if sendErr == nil {
		channel.Recipients = []string{event.Recipient}
		message := notify.Message{Subject: event.Subject, Text: text, FromName: "Deuterium IX"}
		if strings.HasPrefix(event.Kind, "CODE_") {
			message, sendErr = notify.DecodeVerificationPayload(text)
		}
		if event.Kind == "HELP_SUBMISSION" {
			message.FromName = event.ReplyTo + "（Help 转交）"
		}
		if event.Kind == "TICKET_CREATED" {
			message = notify.RenderSupportAcknowledgement(event.Reference, event.Subject, text)
		}
		message.ReplyTo = event.ReplyTo
		message.InReplyTo = event.InReplyTo
		if event.InReplyTo != "" {
			message.References = []string{event.InReplyTo}
		}
		if event.HeadersCipher != "" {
			raw, err := notify.Open(s.Config.SMTPKey, event.HeadersCipher)
			if err != nil {
				sendErr = err
			} else {
				sendErr = json.Unmarshal([]byte(raw), &message.Headers)
			}
		}
		if sendErr == nil {
			sendErr = send(ctx, channel.Settings, password, event.EventID, message)
		}
	}
	finish, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return s.Store.FinishMail(finish, event, sendErr)
}

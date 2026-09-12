package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/notify"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func (s *Server) registerAdminEmailV204(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/admin/email-settings", s.adminEmailGetV204)
	mux.HandleFunc("PUT /api/v1/admin/email-settings", s.adminEmailSaveV204)
	mux.HandleFunc("POST /api/v1/admin/email-settings/test", s.adminEmailTestV204)
	mux.HandleFunc("GET /api/v1/admin/email-settings/templates/{decision}", s.adminEmailTemplate)
}
func (s *Server) adminEmailGetV204(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "platform.admin"); err != nil {
		failError(w, r, err)
		return
	}
	settings, err := s.Store.EmailSettingsV204(r.Context())
	if err != nil {
		failError(w, r, err)
		return
	}
	status, err := s.Store.EmailStatusV204(r.Context())
	if err != nil {
		failError(w, r, err)
		return
	}
	v2Success(w, r, map[string]any{"settings": settings, "delivery": status, "secretStorageReady": len(s.Config.SMTPKey) == 32})
}
func (s *Server) adminEmailSaveV204(w http.ResponseWriter, r *http.Request) {
	u, err := s.admin(r, "platform.admin")
	if err != nil {
		failError(w, r, err)
		return
	}
	var input struct {
		ClientRequestID        string   `json:"clientRequestId"`
		ExpectedVersion        int64    `json:"expectedVersion"`
		Enabled                bool     `json:"enabled"`
		Host                   string   `json:"host"`
		Port                   int      `json:"port"`
		Security               string   `json:"security"`
		Username               string   `json:"username"`
		From                   string   `json:"from"`
		Recipients             []string `json:"recipients"`
		Password               *string  `json:"password,omitempty"`
		AdmissionReviewEnabled *bool    `json:"admissionReviewEnabled,omitempty"`
	}
	if err = body(w, r, &input); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	settings := notify.Settings{Enabled: input.Enabled, Host: input.Host, Port: input.Port, Security: input.Security, Username: input.Username, From: input.From, Recipients: input.Recipients}
	if input.AdmissionReviewEnabled != nil {
		settings.AdmissionReviewEnabled = *input.AdmissionReviewEnabled
	} else {
		previous, loadErr := s.Store.EmailSettingsV204(r.Context())
		if loadErr != nil {
			failError(w, r, loadErr)
			return
		}
		settings.AdmissionReviewEnabled = previous.AdmissionReviewEnabled
	}
	if err = settings.Validate(); err != nil {
		failure(w, r, 400, "EMAIL_SETTINGS_INVALID", err.Error())
		return
	}
	if input.Password != nil && (len(*input.Password) > 2048 || strings.ContainsRune(*input.Password, 0)) {
		failure(w, r, 400, "EMAIL_SETTINGS_INVALID", "邮件密码格式无效。")
		return
	}
	if input.Password != nil && *input.Password != "" && len(s.Config.SMTPKey) != 32 {
		failure(w, r, 503, "EMAIL_KEY_UNAVAILABLE", "服务器邮件密钥尚未配置，暂时无法保存密码。")
		return
	}
	result, err := s.Store.SaveEmailSettingsV204(r.Context(), u.ID, input.ClientRequestID, input.ExpectedVersion, settings, input.Password, s.Config.SMTPKey)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	v2Success(w, r, result)
}
func (s *Server) adminEmailTestV204(w http.ResponseWriter, r *http.Request) {
	u, err := s.admin(r, "platform.admin")
	if err != nil {
		failError(w, r, err)
		return
	}
	var input struct {
		ClientRequestID string `json:"clientRequestId"`
		Template        string `json:"template,omitempty"`
	}
	if err = body(w, r, &input); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	settings, err := s.Store.EmailSettingsV204(r.Context())
	if err != nil {
		failError(w, r, err)
		return
	}
	if !settings.Enabled || settings.Validate() != nil {
		failure(w, r, 409, "EMAIL_NOT_CONFIGURED", "请先保存并启用邮件配置。")
		return
	}
	var result json.RawMessage
	switch input.Template {
	case "", "CONNECTION":
		result, err = s.Store.EnqueueTestEmailV204(r.Context(), u.ID, input.ClientRequestID)
	case "APPROVED", "REJECTED":
		result, err = s.Store.EnqueueAdmissionEmailPreview(r.Context(), u.ID, input.ClientRequestID, input.Template)
	default:
		failure(w, r, 400, "EMAIL_TEMPLATE_INVALID", "请选择连接测试、通过邮件或拒绝邮件。")
		return
	}
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	v2Success(w, r, result)
}

func (s *Server) adminEmailTemplate(w http.ResponseWriter, r *http.Request) {
	u, err := s.admin(r, "platform.admin")
	if err != nil {
		failError(w, r, err)
		return
	}
	decision := r.PathValue("decision")
	if decision != "APPROVED" && decision != "REJECTED" {
		failure(w, r, 404, "EMAIL_TEMPLATE_NOT_FOUND", "邮件模板不存在。")
		return
	}
	m, err := notify.RenderAdmission(notify.AdmissionMail{TemplateVersion: notify.AdmissionTemplateVersion, GameID: u.GameID, Decision: decision, Reason: "申请中的玩家信息与核验结果不一致，请核对正版玩家 ID 后重新提交。", ApplicationURL: s.Config.AdmissionApplicationURL, Preview: true})
	if err != nil {
		failError(w, r, err)
		return
	}
	v2Success(w, r, map[string]any{"subject": m.Subject, "html": notify.AdmissionPreviewHTML(m), "templateVersion": notify.AdmissionTemplateVersion})
}

func (s *Server) emailWorkerV204() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			for n := 0; n < 5; n++ {
				if err := s.processEmailV204(s.ctx, notify.SendMessage); err != nil {
					if !errors.Is(err, sql.ErrNoRows) && !errors.Is(err, context.Canceled) {
						slog.Warn("email delivery worker deferred", "error", err)
					}
					break
				}
			}
		}
	}
}

type emailSenderV204 func(context.Context, notify.Settings, string, string, notify.Message) error

func (s *Server) processEmailV204(ctx context.Context, send emailSenderV204) error {
	settings, err := s.Store.EmailSettingsV204(ctx)
	if err != nil {
		return err
	}
	if !settings.Enabled {
		return sql.ErrNoRows
	}
	event, err := s.Store.ClaimEmailV204(ctx)
	if err != nil {
		return err
	}
	if event.Recipient == "" {
		if err := s.Store.BindEmailRecipientsV204(ctx, event, settings.Recipients); err != nil {
			return err
		}
	} else {
		settings.Recipients = strings.Split(event.Recipient, ", ")
	}
	password, sendError := notify.Open(s.Config.SMTPKey, settings.PasswordCipher)
	if sendError == nil {
		message := notify.Message{Subject: "Deuterium 邮件提醒测试", Text: "邮件服务配置已成功连接。后台已启用的邮件提醒将使用此发送通道。"}
		if event.ApplicationID != nil || event.CaseState == "TEST_APPROVED" || event.CaseState == "TEST_REJECTED" {
			var payload store.AdmissionEmailPayload
			if json.Unmarshal([]byte(event.Payload), &payload) != nil || payload.Decision != strings.TrimPrefix(event.CaseState, "TEST_") {
				sendError = errors.New("白名单邮件记录无法读取。")
			} else {
				var current bool
				current, sendError = s.Store.AdmissionEmailCurrent(ctx, event, payload)
				if sendError == nil && !current {
					return s.Store.CancelEmailV204(ctx, event, "审核结果或通行资格已变化，取消旧通知。")
				}
				if sendError == nil {
					message, sendError = notify.RenderAdmission(notify.AdmissionMail{TemplateVersion: payload.TemplateVersion, GameID: payload.GameID, Decision: payload.Decision, Reason: payload.Reason, ApplicationURL: s.Config.AdmissionApplicationURL, Preview: event.ApplicationID == nil})
				}
			}
		} else if event.CaseID != nil {
			state := map[string]string{"SUBMITTED": "新申请待受理", "IN_REVIEW": "处理中 / 有补充更新", "WAITING_EVIDENCE": "等待补充资料", "RESOLVING": "正在处理资金", "RESOLVED": "已结案", "WITHDRAWN": "已撤回"}[event.CaseState]
			if state == "" {
				state = event.CaseState
			}
			message.Subject = "Deuterium 平台介入 · " + state
			message.Text = fmt.Sprintf("案件：%s\n状态：%s\n更新时间：%s\n\n请登录平台管理页查看并处理：\n%s/admin?section=interventions&case=%s\n\n邮件不包含交易双方提交的证据正文。", *event.CaseID, state, event.CreatedAt.In(time.FixedZone("UTC+8", 8*3600)).Format("2006-01-02 15:04:05"), s.Config.PublicOrigin, url.QueryEscape(*event.CaseID))
		}
		if sendError == nil {
			sendError = send(ctx, settings.Settings, password, store.Digest([]byte(event.EventID))[:40], message)
		}
	}
	finishCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err = s.Store.FinishEmailV204(finishCtx, event, sendError); err != nil {
		return err
	}
	if sendError != nil {
		return errors.New("email delivery deferred")
	}
	return nil
}

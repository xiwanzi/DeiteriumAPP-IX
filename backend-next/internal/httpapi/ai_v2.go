package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

type aiGatewayV2 struct {
	server      *Server
	config      aiConfigV2
	configError error
	client      *http.Client
	active      *atomic.Int32
}

func (s *Server) registerAIV2(mux *http.ServeMux) {
	c, err := loadAIConfigV2()
	s.registerAIConfigV2(mux, c, err)
}
func (s *Server) registerAIConfigV2(mux *http.ServeMux, c aiConfigV2, configError error) *aiGatewayV2 {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 45 * time.Second
	transport.MaxIdleConnsPerHost = max(1, int(c.MaxConcurrent))
	g := &aiGatewayV2{server: s, config: c, configError: configError, client: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, active: &atomic.Int32{}}
	mux.HandleFunc("GET /api/v1/ai/me", g.configuredV206((*aiGatewayV2).me))
	mux.HandleFunc("GET /api/v1/ai/plans", g.configuredV206((*aiGatewayV2).plans))
	mux.HandleFunc("GET /api/v1/ai/messages", g.configuredV206((*aiGatewayV2).messages))
	mux.HandleFunc("POST /api/v1/ai/conversation/reset", g.configuredV206((*aiGatewayV2).reset))
	mux.HandleFunc("POST /api/v1/ai/chat/stream", g.configuredV206((*aiGatewayV2).stream))
	mux.HandleFunc("POST /api/v1/ai/purchase-quotes", g.configuredV206((*aiGatewayV2).quoteV207))
	mux.HandleFunc("POST /api/v1/ai/purchases", g.configuredV206((*aiGatewayV2).purchaseV206))
	mux.HandleFunc("GET /api/v1/ai/purchases/{purchaseId}", g.purchaseGetV206)
	g.registerSettingsV206(mux)
	return g
}
func (g *aiGatewayV2) authenticate(w http.ResponseWriter, r *http.Request) (store.Session, bool) {
	u, err := g.server.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return u, false
	}
	if !g.config.Enabled || g.configError != nil {
		failure(w, r, 503, "AI_DISABLED", "AI 服务暂不可用，请稍后重试。 ")
		return u, false
	}
	return u, true
}
func aiFailureV2(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, store.ErrAIQuotaV2):
		failure(w, r, 429, "AI_QUOTA_EXCEEDED", "本时段 AI 额度已用完，请等待恢复。")
	case errors.Is(err, store.ErrAIBusyV2):
		failure(w, r, 409, "AI_REQUEST_IN_PROGRESS", "上一条 AI 回复仍在生成，可使用原请求继续查看。")
	case errors.Is(err, store.ErrAIConversationV2):
		failure(w, r, 409, "AI_CONVERSATION_CHANGED", "请求属于旧会话，请在当前会话重新发送。")
	case errors.Is(err, store.ErrAINotFoundV2):
		failure(w, r, 404, "AI_MESSAGE_NOT_FOUND", "AI 消息不存在或不可见。")
	case errors.Is(err, store.ErrConflict):
		failure(w, r, 409, "AI_REQUEST_CONFLICT", "相同请求标识不能发送不同内容。")
	default:
		failError(w, r, err)
	}
}
func (g *aiGatewayV2) state(ctx context.Context, user string) (store.AIStateV2, error) {
	return g.server.Store.AIStateV2(ctx, user, g.config.policy(), time.Now().UTC())
}
func (g *aiGatewayV2) me(w http.ResponseWriter, r *http.Request) {
	u, ok := g.authenticate(w, r)
	if !ok {
		return
	}
	state, err := g.state(r.Context(), u.User.ID)
	if err != nil {
		aiFailureV2(w, r, err)
		return
	}
	success(w, r, map[string]any{"assistantName": g.config.AssistantName, "plan": state.Plan, "expiresAt": state.ExpiresAt, "quota": state.Quota, "conversation": state.Conversation, "pendingRequest": state.Pending, "maxInputChars": int(g.config.MaxInput), "webSearchAvailable": g.config.WebSearch})
}
func (g *aiGatewayV2) plans(w http.ResponseWriter, r *http.Request) {
	if _, ok := g.authenticate(w, r); !ok {
		return
	}
	plans, err := g.server.Store.AIPlansV2(r.Context(), g.config.policy())
	if err != nil {
		aiFailureV2(w, r, err)
		return
	}
	success(w, r, map[string]any{"plans": plans})
}
func (g *aiGatewayV2) messages(w http.ResponseWriter, r *http.Request) {
	u, ok := g.authenticate(w, r)
	if !ok {
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			failure(w, r, 400, "INVALID_REQUEST", "消息数量应为 1–100。 ")
			return
		}
		limit = n
	}
	before := r.URL.Query().Get("beforeMessageId")
	if len(before) > 64 {
		failure(w, r, 400, "INVALID_REQUEST", "消息游标不正确。 ")
		return
	}
	state, err := g.state(r.Context(), u.User.ID)
	if err != nil {
		aiFailureV2(w, r, err)
		return
	}
	messages, err := g.server.Store.AIMessagesV2(r.Context(), u.User.ID, state.Conversation.ID, before, limit+1)
	if err != nil {
		aiFailureV2(w, r, err)
		return
	}
	hasMore := len(messages) > limit
	var next any
	if hasMore {
		messages = messages[1:]
		next = messages[0].ID
	}
	writeJSON(w, 200, map[string]any{"requestId": requestID(r), "data": map[string]any{"messages": messages}, "page": map[string]any{"nextCursor": next, "hasMore": hasMore}})
}
func (g *aiGatewayV2) reset(w http.ResponseWriter, r *http.Request) {
	u, ok := g.authenticate(w, r)
	if !ok {
		return
	}
	var input struct{}
	if err := body(w, r, &input); err != nil {
		failure(w, r, 400, "INVALID_REQUEST", "新对话请求不正确。 ")
		return
	}
	conversation, err := g.server.Store.ResetAIV2(r.Context(), u.User.ID, time.Now().UTC())
	if err != nil {
		aiFailureV2(w, r, err)
		return
	}
	success(w, r, map[string]any{"conversation": conversation})
}
func (g *aiGatewayV2) run(e store.AIExchangeV2) {
	defer g.active.Add(-1)
	ctx, cancel := context.WithTimeout(g.server.ctx, time.Duration(g.config.TimeoutSeconds)*time.Second)
	defer cancel()
	history, err := g.server.Store.AIContextV2(ctx, e, int(g.config.MaxContext))
	result := aiProviderResultV2{Status: "failed", Code: "AI_PROVIDER_UNAVAILABLE"}
	checkpoint := func() error {
		checkCtx, stop := context.WithTimeout(ctx, 5*time.Second)
		defer stop()
		return g.server.Store.CheckpointAIV2(checkCtx, e)
	}
	if err == nil {
		result = runAIProviderV2(ctx, g.client, g.config, &e, history, checkpoint)
	}
	finishCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer stop()
	// If this write cannot be confirmed, the persisted deadline later fences the request UNKNOWN.
	_ = g.server.Store.FinishAIV2(finishCtx, e, result.Status, result.Code, result.Reason, result.InputTokens, result.OutputTokens)
}
func aiEventV2(w http.ResponseWriter, name string, value any) error {
	controller := http.NewResponseController(w)
	_ = controller.SetWriteDeadline(time.Now().Add(15 * time.Second))
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, encoded); err != nil {
		return err
	}
	return controller.Flush()
}
func aiErrorEventV2(w http.ResponseWriter, code, message string) error {
	return aiEventV2(w, "error", map[string]any{"error": map[string]any{"code": code, "message": message, "retryAfterSeconds": 2}})
}
func aiBodyV2(w http.ResponseWriter, r *http.Request, value any) error {
	if strings.TrimSpace(strings.ToLower(strings.Split(r.Header.Get("Content-Type"), ";")[0])) != "application/json" {
		return errors.New("invalid ai content type")
	}
	// 2,000 escaped surrogate-pair emoji can take 24 KiB in otherwise valid JSON.
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errors.New("trailing ai request data")
	}
	return nil
}
func (g *aiGatewayV2) stream(w http.ResponseWriter, r *http.Request) {
	u, ok := g.authenticate(w, r)
	if !ok {
		return
	}
	var input struct {
		ClientMessageID string `json:"clientMessageId"`
		Content         string `json:"content"`
	}
	if err := aiBodyV2(w, r, &input); err != nil || strings.TrimSpace(input.ClientMessageID) == "" || len(input.ClientMessageID) > 128 || !utf8.ValidString(input.Content) || utf8.RuneCountInString(input.Content) > int(g.config.MaxInput) || strings.TrimSpace(input.Content) == "" {
		failure(w, r, 400, "AI_INVALID_MESSAGE", "请输入有效内容，且不超过允许字数。 ")
		return
	}
	input.Content = strings.TrimSpace(input.Content)
	if input.Content == "/new" {
		conversation, err := g.server.Store.ResetAIV2(r.Context(), u.User.ID, time.Now().UTC())
		if err != nil {
			aiFailureV2(w, r, err)
			return
		}
		state, err := g.state(r.Context(), u.User.ID)
		if err != nil {
			aiFailureV2(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("X-Accel-Buffering", "no")
		w.Header().Set("Cache-Control", "no-store")
		if aiEventV2(w, "meta", map[string]any{"conversationId": conversation.ID, "userMessageId": nil, "assistantMessageId": nil, "quota": state.Quota}) != nil {
			return
		}
		_ = aiEventV2(w, "done", map[string]any{"quota": state.Quota, "conversation": conversation})
		return
	}
	e, created, err := g.server.Store.BeginAIV2(r.Context(), u.User.ID, input.ClientMessageID, input.Content, g.config.Model, g.config.policy(), time.Now().UTC(), time.Duration(g.config.TimeoutSeconds+30)*time.Second)
	if err != nil {
		aiFailureV2(w, r, err)
		return
	}
	if created {
		if g.active.Add(1) <= int32(g.config.MaxConcurrent) {
			go g.run(e)
		} else {
			g.active.Add(-1)
			ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
			_ = g.server.Store.FinishAIV2(ctx, e, "failed", "AI_SERVER_BUSY", "", 0, 0)
			stop()
		}
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Cache-Control", "no-store")
	state, err := g.state(r.Context(), u.User.ID)
	if err != nil {
		_ = aiErrorEventV2(w, "AI_RESULT_UNKNOWN", "请求已受理，但状态暂时无法读取，请使用原请求恢复。")
		return
	}
	if aiEventV2(w, "meta", map[string]any{"conversationId": e.ConversationID, "userMessageId": e.UserMessageID, "assistantMessageId": e.AssistantMessageID, "userMessage": store.AIUserMessageV2(e), "quota": state.Quota}) != nil {
		return
	}
	poll := time.NewTicker(250 * time.Millisecond)
	defer poll.Stop()
	heartbeat := time.NewTicker(10 * time.Second)
	defer heartbeat.Stop()
	position := 0
	lastSources, lastProgress := "", ""
	lastAuth := time.Time{}
	for {
		if time.Since(lastAuth) > time.Second {
			if _, err = g.server.Store.Session(r.Context(), u.TokenHash); err != nil {
				_ = aiErrorEventV2(w, "UNAUTHORIZED", "登录状态已失效，请重新登录。")
				return
			}
			lastAuth = time.Now()
		}
		e, err = g.server.Store.AIExchangeV2(r.Context(), u.User.ID, e.ID)
		if err != nil {
			_ = aiErrorEventV2(w, "AI_RESULT_UNKNOWN", "暂时无法读取生成结果，请使用原请求恢复。")
			return
		}
		if time.Now().After(e.DeadlineAt) && (e.Status == "pending" || e.Status == "streaming") {
			if _, err = g.state(r.Context(), u.User.ID); err != nil {
				_ = aiErrorEventV2(w, "AI_RESULT_UNKNOWN", "请求状态待确认。")
				return
			}
			continue
		}
		if e.Progress != lastProgress {
			if aiEventV2(w, "status", map[string]string{"status": e.Progress}) != nil {
				return
			}
			lastProgress = e.Progress
		}
		if len(e.Answer) > position {
			if aiEventV2(w, "delta", map[string]string{"content": e.Answer[position:]}) != nil {
				return
			}
			position = len(e.Answer)
		}
		encoded, _ := json.Marshal(e.Sources)
		if string(encoded) != lastSources && len(e.Sources) > 0 {
			if aiEventV2(w, "sources", map[string]any{"sources": e.Sources}) != nil {
				return
			}
			lastSources = string(encoded)
		}
		switch e.Status {
		case "completed":
			state, err = g.state(r.Context(), u.User.ID)
			if err != nil {
				_ = aiErrorEventV2(w, "AI_RESULT_UNKNOWN", "回复已生成，状态读取待恢复。")
				return
			}
			_ = aiEventV2(w, "done", map[string]any{"message": store.AIAssistantMessageV2(e), "quota": state.Quota})
			return
		case "failed":
			message := "AI 服务暂不可用，本次未消耗额度。"
			if e.ErrorCode == "AI_SERVER_BUSY" {
				message = "AI 服务繁忙，请稍后重新发送。"
			}
			_ = aiErrorEventV2(w, e.ErrorCode, message)
			return
		case "unknown", "incomplete":
			code := e.ErrorCode
			if code == "" {
				code = "AI_RESULT_UNKNOWN"
			}
			_ = aiErrorEventV2(w, code, "这条回复尚未完整完成，已保留现有内容。原请求不会重复生成；可输入 /new 开始新对话。")
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-g.server.ctx.Done():
			return
		case <-poll.C:
		case <-heartbeat.C:
			controller := http.NewResponseController(w)
			_ = controller.SetWriteDeadline(time.Now().Add(15 * time.Second))
			if _, err = fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
				return
			}
			if controller.Flush() != nil {
				return
			}
		}
	}
}

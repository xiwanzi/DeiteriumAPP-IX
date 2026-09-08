package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func sendSocket(ctx context.Context, c *websocket.Conn, kind, requestID string, payload any) error {
	frame := map[string]any{"type": kind, "sentAt": time.Now().UTC().Format(time.RFC3339Nano), "payload": payload}
	if requestID != "" {
		frame["requestId"] = requestID
	}
	data, err := json.Marshal(frame)
	if err != nil {
		return err
	}
	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return c.Write(writeCtx, websocket.MessageText, data)
}
func readSocket(ctx context.Context, c *websocket.Conn) (e bridge.Envelope, err error) {
	kind, data, err := c.Read(ctx)
	if err != nil {
		return
	}
	if kind != websocket.MessageText {
		return e, bridge.ErrProtocol
	}
	err = bridge.Decode(data, &e)
	if len(e.RequestID) > 128 || len(e.SentAt) > 40 || len(e.Type) > 64 {
		err = bridge.ErrProtocol
	}
	return
}
func pingSocket(ctx context.Context, c *websocket.Conn) error {
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return c.Ping(pingCtx)
}

func (s *Server) coreSocket(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" || r.Header.Get("Origin") != "" || r.Header.Get("Cookie") != "" {
		failError(w, r, ErrForbidden)
		return
	}
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		failError(w, r, store.ErrUnauthorized)
		return
	}
	node, ok := s.Config.AuthenticateNode(r.Header.Get("X-Deuterium-Node-ID"), strings.TrimPrefix(auth, "Bearer "))
	if !ok {
		failError(w, r, store.ErrUnauthorized)
		return
	}
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer c.CloseNow()
	c.SetReadLimit(32768)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	stop := context.AfterFunc(s.ctx, cancel)
	defer stop()
	helloCtx, helloCancel := context.WithTimeout(ctx, 5*time.Second)
	hello, err := readSocket(helloCtx, c)
	helloCancel()
	var version struct {
		ProtocolVersion int `json:"protocolVersion"`
	}
	if err != nil || hello.Type != "core.hello" || bridge.Decode(hello.Payload, &version) != nil || version.ProtocolVersion != 1 {
		_ = c.Close(websocket.StatusPolicyViolation, "protocol version required")
		return
	}
	if !s.Hub.JoinNode(node.ID) {
		_ = c.Close(websocket.StatusPolicyViolation, "node already connected")
		return
	}
	defer s.Hub.LeaveNode(node.ID)
	s.Core.Connect(node.ID, func(sendCtx context.Context, kind string, payload any) error {
		return sendSocket(sendCtx, c, kind, "", payload)
	})
	defer func() {
		s.Core.Disconnect(node.ID)
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Store.DisconnectCore(cleanup, node.ID)
	}()
	if sendSocket(ctx, c, "core.welcome", hello.RequestID, map[string]any{"protocolVersion": 1, "serverId": node.ID, "chat": node.Chat, "itemPrefix": node.ItemPrefix, "heartbeatSeconds": 20, "maxFrameBytes": 32768, "features": []string{"core-v1", "commands", "catalog", "mailbox-v1", "presence"}}) != nil {
		return
	}
	wake, unsubscribe := s.Hub.Subscribe()
	defer unsubscribe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer cancel()
		for {
			e, err := readSocket(ctx, c)
			if err != nil {
				return
			}
			if handled, err := s.coreSpecial(ctx, node, c, e); handled {
				if err != nil {
					return
				}
				continue
			}
			if e.Type == "chat.delivery.ack" {
				var a struct {
					MessageID string `json:"messageId"`
				}
				if !node.Chat || bridge.Decode(e.Payload, &a) != nil || !bridge.ValidMessageID(a.MessageID) {
					return
				}
				if err = s.Store.AckChat(ctx, node.ID, a.MessageID); err != nil {
					return
				}
				if sendSocket(ctx, c, "chat.delivery.ack.result", e.RequestID, map[string]any{"messageId": a.MessageID, "acknowledged": true}) != nil {
					return
				}
				continue
			}
			payload, chat, item, err := bridge.ValidateCore(s.Config, node, e)
			if err != nil {
				_ = sendSocket(ctx, c, "core.event.result", e.RequestID, map[string]any{"eventId": e.EventID, "status": "rejected", "error": map[string]string{"code": "INVALID_EVENT", "message": "事件格式或能力权限不符。"}})
				continue
			}
			seq, replay, err := s.Store.CoreEvent(ctx, node.ID, e.EventID, e.Type, payload, chat, item)
			if err != nil {
				status, code := "unknown", "SERVICE_UNAVAILABLE"
				if errors.Is(err, store.ErrConflict) {
					status, code = "rejected", "IDEMPOTENCY_CONFLICT"
				}
				if sendSocket(ctx, c, "core.event.result", e.RequestID, map[string]any{"eventId": e.EventID, "status": status, "error": map[string]string{"code": code, "message": "请保留原事件标识查询或重投。"}}) != nil {
					return
				}
				continue
			}
			if !replay {
				s.Hub.Wake()
			}
			if chat != nil {
				_ = s.Store.PublicSocialNotificationsV2(ctx, "core:"+node.ID, e.EventID)
			}
			if sendSocket(ctx, c, "core.event.result", e.RequestID, map[string]any{"eventId": e.EventID, "status": "committed", "sequence": seq, "replayed": replay}) != nil {
				return
			}
		}
	}()
	defer func() { cancel(); c.CloseNow(); <-done }()
	poll := time.NewTicker(5 * time.Second)
	defer poll.Stop()
	ping := time.NewTicker(20 * time.Second)
	defer ping.Stop()
	select {
	case wake <- struct{}{}:
	default:
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ping.C:
			if pingSocket(ctx, c) != nil {
				return
			}
			continue
		case <-poll.C:
		case <-wake:
		}
		if !node.Chat {
			continue
		}
		pending, err := s.Store.PendingChats(ctx, node.ID)
		if err != nil {
			return
		}
		for _, m := range pending {
			if sendSocket(ctx, c, "chat.app.delivery", "", map[string]any{"messageId": m.ID, "senderUuid": m.ServerUUID, "gameId": m.Sender.GameID, "content": m.Content, "origin": "app", "expiresAt": m.SentAt.Add(10 * time.Minute), "suppressRebroadcast": true}) != nil {
				return
			}
		}
	}
}

func (s *Server) appSocket(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	session, err := s.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	if (session.Kind == "web" && !s.origin(r)) || (r.Header.Get("Origin") != "" && !s.origin(r)) {
		failError(w, r, ErrForbidden)
		return
	}
	cursor, err := s.Store.LatestChatSequence(r.Context())
	if err != nil {
		failError(w, r, err)
		return
	}
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{s.Config.PublicOrigin}})
	if err != nil {
		return
	}
	defer c.CloseNow()
	c.SetReadLimit(4096)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	stop := context.AfterFunc(s.ctx, cancel)
	defer stop()
	wake, unsubscribe := s.Hub.Subscribe()
	defer unsubscribe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer cancel()
		for {
			e, err := readSocket(ctx, c)
			if err != nil {
				return
			}
			if e.Type != "chat.send" {
				_ = sendSocket(ctx, c, "error", e.RequestID, map[string]any{"code": "CAPABILITY_UNAVAILABLE", "message": "当前版本尚未实现该实时能力。"})
				continue
			}
			result := s.sendChat(ctx, session.TokenHash, e.Payload)
			if sendSocket(ctx, c, "chat.send.result", e.RequestID, result) != nil {
				return
			}
		}
	}()
	defer func() { cancel(); c.CloseNow(); <-done }()
	poll := time.NewTicker(5 * time.Second)
	defer poll.Stop()
	ping := time.NewTicker(20 * time.Second)
	defer ping.Stop()
	// Query once immediately: covers commits between cursor lookup and subscription.
	select {
	case wake <- struct{}{}:
	default:
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ping.C:
			if pingSocket(ctx, c) != nil {
				return
			}
			continue
		case <-poll.C:
		case <-wake:
		}
		if _, err = s.Store.Session(ctx, session.TokenHash); err != nil {
			_ = c.Close(websocket.StatusPolicyViolation, "session expired or revoked")
			return
		}
		messages, err := s.Store.Messages(ctx, cursor, true, 50)
		if err != nil {
			return
		}
		for _, m := range messages {
			if sendSocket(ctx, c, "chat.message", "", map[string]any{"message": m}) != nil {
				return
			}
			cursor = m.Sequence
		}
		if len(messages) == 50 {
			select {
			case wake <- struct{}{}:
			default:
			}
		}
	}
}

func (s *Server) sendChat(ctx context.Context, tokenHash string, payload []byte) map[string]any {
	var in struct {
		ClientMessageID     string   `json:"clientMessageId"`
		Content             string   `json:"content"`
		MentionedPlayerRefs []string `json:"mentionedPlayerRefs"`
		ReplyToMessageID    string   `json:"replyToMessageId"`
	}
	failed := func(code, message string) map[string]any {
		return map[string]any{"clientMessageId": in.ClientMessageID, "status": "failed", "error": map[string]string{"code": code, "message": message}}
	}
	if bridge.Decode(payload, &in) != nil || !bridge.ValidMessageID(in.ClientMessageID) {
		return failed("INVALID_REQUEST", "消息格式不正确。")
	}
	content, err := bridge.Content(in.Content)
	if err != nil {
		return failed("INVALID_REQUEST", "消息长度或内容不符合要求。")
	}
	v, err := s.Store.Session(ctx, tokenHash)
	if err != nil {
		return failed("UNAUTHORIZED", "会话已失效。")
	}
	id, err := s.Store.ExistingAppChatV2(ctx, v.User.ID, in.ClientMessageID, content, in.ReplyToMessageID, in.MentionedPlayerRefs)
	if err == nil {
		_ = s.Store.PublicSocialNotificationsV2(ctx, "app:"+v.User.ID, in.ClientMessageID)
		return map[string]any{"clientMessageId": in.ClientMessageID, "status": "accepted", "messageId": id}
	}
	if errors.Is(err, store.ErrConflict) {
		return failed("IDEMPOTENCY_CONFLICT", "相同消息标识对应不同内容。")
	}
	if errors.Is(err, store.ErrSocialInvalid) {
		return failed("INVALID_REQUEST", "引用或提及格式不正确。")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return failed("SERVICE_UNAVAILABLE", "服务暂不可用。")
	}
	nodes := s.Config.ChatNodes()
	if !s.Hub.AnyNode(nodes) {
		return failed("PLUGIN_BRIDGE_UNAVAILABLE", "服务器连接暂不可用，请稍后再试。")
	}
	if err = s.Store.ReserveChat(ctx, v.User.ID); err != nil {
		if errors.Is(err, store.ErrRateLimited) {
			return failed("RATE_LIMITED", "发送过于频繁，请稍后再试。")
		}
		return failed("SERVICE_UNAVAILABLE", "服务暂不可用。")
	}
	id, _, err = s.Store.PublishAppChatV2(ctx, v.User, in.ClientMessageID, content, in.ReplyToMessageID, in.MentionedPlayerRefs, nodes)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			return failed("IDEMPOTENCY_CONFLICT", "相同消息标识对应不同内容。")
		}
		if errors.Is(err, store.ErrSocialNotFound) {
			return failed("NOT_FOUND", "原消息或提及玩家不存在或不可见。")
		}
		if errors.Is(err, store.ErrSocialInvalid) {
			return failed("INVALID_REQUEST", "引用或提及格式不正确。")
		}
		return failed("RESULT_UNKNOWN", "请使用原消息标识重试确认。")
	}
	_ = s.Store.PublicSocialNotificationsV2(ctx, "app:"+v.User.ID, in.ClientMessageID)
	s.Hub.Wake()
	return map[string]any{"clientMessageId": in.ClientMessageID, "status": "accepted", "messageId": id}
}

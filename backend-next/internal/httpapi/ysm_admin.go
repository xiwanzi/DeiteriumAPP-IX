package httpapi

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
)

const ysmPrefix = "/api/v1/admin/ysm/"
const ysmUploadLimit int64 = 512 << 20
const ysmResponseLimit int64 = 16 << 20

var ysmRoutes = map[string]string{
	"config": "GET", "cluster": "GET", "dashboard": "GET", "catalog": "GET",
	"tasks": "GET", "diff": "GET", "audit": "GET", "migration": "GET", "gc": "GET", "diagnostics.zip": "GET",
	"operations/preview": "POST", "operations/confirm": "POST", "models/stage": "POST",
	"models/delete/preview": "POST", "models/confirm": "POST", "config/preview": "POST", "config/confirm": "POST",
}

var ysmCloudModelRoute = regexp.MustCompile(`^cloud/models/[A-Za-z0-9._-]+/(versions|archive|restore|rollback)$`)
var ysmCloudNPCRoute = regexp.MustCompile(`^cloud/npcs/[A-Za-z0-9._-]+/[a-fA-F0-9-]{36}$`)

func ysmAuditAction(path string) string {
	if match := ysmCloudModelRoute.FindStringSubmatch(path); match != nil {
		return "ysm.cloud.models." + match[1]
	}
	if ysmCloudNPCRoute.MatchString(path) {
		return "ysm.cloud.npcs.update"
	}
	return "ysm." + strings.ReplaceAll(path, "/", ".")
}

func ysmRouteMethod(path, requested string) (string, bool) {
	if path == "cloud/connection" {
		return requested, requested == "GET" || requested == "POST"
	}
	switch path {
	case "cloud/overview", "cloud/models", "cloud/releases", "cloud/players", "cloud/npcs", "cloud/activity":
		return "GET", true
	case "cloud/upload":
		return "POST", true
	}
	if match := ysmCloudModelRoute.FindStringSubmatch(path); match != nil {
		if match[1] == "versions" {
			return "GET", true
		}
		return "POST", true
	}
	if ysmCloudNPCRoute.MatchString(path) {
		return "POST", true
	}
	method, exists := ysmRoutes[path]
	return method, exists
}

type ysmBridge struct {
	url, origin, host string
	client            *http.Client
	seal              cipher.AEAD
	slots             chan struct{}
}

// A sealed confirmation retains the upstream CSRF without exposing it or sharing
// it between logins. The upstream still owns expiry and one-time consumption.
type ysmConfirmation struct {
	Token   string `json:"token"`
	CSRF    string `json:"csrf"`
	Expires int64  `json:"expires"`
}

func newYSMBridge(c config.Config) *ysmBridge {
	if c.YSMAdminURL == "" || c.ValidateYSM() != nil {
		return nil
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil
	}
	block, _ := aes.NewCipher(key)
	seal, _ := cipher.NewGCM(block)
	origin, _ := url.Parse(c.YSMOrigin())
	return &ysmBridge{url: c.YSMAdminURL, origin: c.YSMOrigin(), host: origin.Host, seal: seal, slots: make(chan struct{}, 4), client: &http.Client{
		Timeout:       5 * time.Minute,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		Transport:     &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext, ResponseHeaderTimeout: 5 * time.Minute, MaxConnsPerHost: 4, MaxIdleConnsPerHost: 4, IdleConnTimeout: 30 * time.Second, MaxResponseHeaderBytes: 16384},
	}}
}

func (s *Server) registerYSM(mux *http.ServeMux) { mux.HandleFunc(ysmPrefix, s.ysmAdmin) }

func (s *Server) ysmAdmin(w http.ResponseWriter, r *http.Request) {
	session, err := s.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	allowed, err := s.Store.HasPermission(r.Context(), session.User.ID, "platform.admin")
	if err != nil {
		failError(w, r, err)
		return
	}
	if !allowed {
		failError(w, r, ErrForbidden)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, ysmPrefix)
	method, exists := ysmRouteMethod(path, r.Method)
	if !exists || r.URL.RawQuery != "" || r.URL.RawPath != "" {
		failure(w, r, 404, "NOT_FOUND", "YSM 接口不存在。")
		return
	}
	if r.Method != method {
		failure(w, r, 405, "METHOD_NOT_ALLOWED", "此接口不支持该操作。")
		return
	}
	b := s.ysm
	if b == nil {
		failure(w, r, 503, "YSM_NOT_CONFIGURED", "YSM 管理连接尚未配置。")
		return
	}
	select {
	case b.slots <- struct{}{}:
		defer func() { <-b.slots }()
	default:
		failure(w, r, 503, "YSM_BUSY", "YSM 管理任务较多，请稍后重试。")
		return
	}
	confirm := strings.HasSuffix(path, "/confirm")
	cloudMutation := strings.HasPrefix(path, "cloud/") && r.Method == "POST"
	ctx := r.Context()
	timeout := 15 * time.Second
	if r.Method == "POST" {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	controller := http.NewResponseController(w)
	// Only this authenticated, concurrency-limited surface needs long uploads/migrations.
	_ = controller.SetReadDeadline(time.Now().Add(timeout))
	_ = controller.SetWriteDeadline(time.Now().Add(timeout + 5*time.Second))
	var body io.Reader
	csrf := ""
	if r.Method == "POST" {
		limit := int64(1 << 20)
		if path == "models/stage" {
			limit = ysmUploadLimit
		}
		if path == "cloud/upload" {
			limit = 128 << 20
			if r.ContentLength <= 0 {
				failure(w, r, 411, "LENGTH_REQUIRED", "上传文件必须具有明确大小。")
				return
			}
		}
		if r.ContentLength > limit {
			failure(w, r, 413, "YSM_UPLOAD_TOO_LARGE", "文件超过上传大小限制。")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, limit)
		if confirm {
			var input struct {
				ConfirmationToken string `json:"confirmationToken"`
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			var extra any
			if decoder.Decode(&input) != nil || decoder.Decode(&extra) != io.EOF {
				failure(w, r, 400, "INVALID_REQUEST", "确认请求格式不正确。")
				return
			}
			ticket, err := b.openConfirmation(input.ConfirmationToken, session.TokenHash, path)
			if err != nil {
				failure(w, r, 409, "YSM_CONFIRMATION_EXPIRED", "确认已失效，请重新预览。")
				return
			}
			csrf = ticket.CSRF
			raw, _ := json.Marshal(map[string]string{"confirmationToken": ticket.Token})
			body = bytes.NewReader(raw)
			// Persist who requested a change before allowing the upstream mutation.
			_, err = s.Store.DB.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,?,?,UTC_TIMESTAMP(6))", session.User.ID, ysmAuditAction(path)+".requested", requestID(r))
			if err != nil {
				failError(w, r, err)
				return
			}
		} else {
			csrf, err = b.csrf(ctx)
			if err != nil {
				failure(w, r, 502, "YSM_UNAVAILABLE", "暂时无法连接 YSM 管理服务。")
				return
			}
			body = r.Body
		}
		if cloudMutation {
			_, err = s.Store.DB.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,?,?,UTC_TIMESTAMP(6))", session.User.ID, ysmAuditAction(path)+".requested", requestID(r)+":"+path)
			if err != nil {
				failError(w, r, err)
				return
			}
		}
	}
	headers := r.Header.Clone()
	if path == "cloud/upload" {
		headers.Set("Content-Length", strconv.FormatInt(r.ContentLength, 10))
	}
	response, err := b.request(ctx, path, r.Method, csrf, body, headers)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			failure(w, r, 413, "YSM_UPLOAD_TOO_LARGE", "文件超过上传大小限制。")
			return
		}
		if confirm || cloudMutation {
			failure(w, r, 504, "YSM_RESULT_UNKNOWN", "等待结果中断，请先查看任务、配置版本及审计记录，不要重复提交。")
		} else {
			failure(w, r, 502, "YSM_UNAVAILABLE", "暂时无法连接 YSM 管理服务，请稍后重试。")
		}
		return
	}
	raw, err := limitedBody(response, ysmResponseLimit)
	if err != nil {
		failure(w, r, 502, "YSM_INVALID_RESPONSE", "YSM 响应未能完整读取；已提交的操作请先核对任务和版本。")
		return
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		status := response.StatusCode
		if status < 400 || status >= 500 {
			status = 502
		}
		message := "YSM 未能完成操作，请查看任务和审计记录。"
		switch status {
		case 403:
			message = "YSM 请求凭据已失效，请重新预览。"
		case 409:
			message = "模型、配置版本或确认已变化，请刷新后重新预览。"
		case 413:
			message = "文件超过 YSM 配置的上传限制。"
		case 400, 415, 422:
			message = "YSM 校验未通过，请核对模型 ID、文件格式或配置内容。"
		case 404:
			message = "YSM 模型或接口不存在，请刷新后重试。"
		}
		code := "YSM_REQUEST_REJECTED"
		if strings.HasPrefix(path, "cloud/") {
			var cloud struct {
				Code string `json:"code"`
			}
			if json.Unmarshal(raw, &cloud) == nil {
				switch cloud.Code {
				case "REVISION_CONFLICT", "ASSET_HASH_MISMATCH", "ASSET_TOO_LARGE", "STORAGE_UNAVAILABLE", "INVALID_METADATA", "IDEMPOTENCY_CONFLICT":
					code = cloud.Code
				}
			}
		}
		failure(w, r, status, code, message)
		return
	}
	if path == "diagnostics.zip" {
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="deuteriumysm-diagnostics.zip"`)
		w.Write(raw)
		return
	}
	var object map[string]json.RawMessage
	if !json.Valid(raw) {
		failure(w, r, 502, "YSM_INVALID_RESPONSE", "YSM 返回格式不正确；请先核对操作结果。")
		return
	}
	if json.Unmarshal(raw, &object) == nil && object != nil {
		delete(object, "csrfToken")
		if tokenRaw, ok := object["confirmationToken"]; ok {
			var token string
			var expires int64
			if json.Unmarshal(tokenRaw, &token) != nil || json.Unmarshal(object["expiresAtEpochSecond"], &expires) != nil || token == "" || csrf == "" {
				failure(w, r, 502, "YSM_INVALID_RESPONSE", "YSM 确认信息不完整，请重新预览。")
				return
			}
			kind := strings.Split(path, "/")[0] + "/confirm"
			ticket, err := b.sealConfirmation(ysmConfirmation{token, csrf, expires}, session.TokenHash, kind)
			if err != nil {
				failure(w, r, 503, "YSM_UNAVAILABLE", "无法建立确认，请重新预览。")
				return
			}
			object["confirmationToken"], _ = json.Marshal(ticket)
		}
		raw, _ = json.Marshal(object)
	}
	if confirm || cloudMutation {
		// Linking the returned task/revision to the request makes the central audit useful
		// while Velocity's original audit remains authoritative for final task outcomes.
		resource := requestID(r)
		for _, key := range []string{"taskId", "configRevision"} {
			if v := object[key]; len(v) > 0 && len(v) < 100 {
				resource += ":" + string(v)
			}
		}
		_, auditErr := s.Store.DB.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,?,?,UTC_TIMESTAMP(6))", session.User.ID, ysmAuditAction(path)+".accepted", resource)
		if auditErr != nil {
			w.Header().Set("X-YSM-Audit-Status", "requested-only")
		}
	}
	success(w, r, json.RawMessage(raw))
}

func (b *ysmBridge) request(ctx context.Context, path, method, csrf string, body io.Reader, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, b.url+"/api/"+path, body)
	if err != nil {
		return nil, err
	}
	req.Host = b.host
	req.Header.Set("Accept", "application/json")
	if method == "POST" {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", b.origin)
		req.Header.Set("Referer", b.origin+"/")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		req.Header.Set("X-CSRF-Token", csrf)
		req.AddCookie(&http.Cookie{Name: "dysm_csrf", Value: csrf})
		if path == "models/stage" {
			for _, key := range []string{"Content-Type", "X-Model-Id", "X-Model-Id-Encoding", "X-Model-Operation", "X-Expected-Revision"} {
				req.Header.Set(key, headers.Get(key))
			}
		}
		if path == "cloud/upload" {
			length, err := strconv.ParseInt(headers.Get("Content-Length"), 10, 64)
			if err != nil || length < 1 || length > 128<<20 {
				return nil, errors.New("invalid Cloud upload length")
			}
			req.ContentLength = length
			for _, key := range []string{"Content-Type", "Idempotency-Key", "X-Asset-Id", "X-Asset-Name", "X-Asset-Format", "X-Asset-Sha256", "X-Asset-Visibility", "X-Asset-Metadata-Encoding", "X-Asset-Base-Revision", "X-Deuterium-Model-Metadata"} {
				req.Header.Set(key, headers.Get(key))
			}
		}
	}
	return b.client.Do(req)
}

func (b *ysmBridge) csrf(ctx context.Context) (string, error) {
	response, err := b.request(ctx, "config", "GET", "", nil, nil)
	if err != nil {
		return "", err
	}
	raw, err := limitedBody(response, 1<<20)
	if err != nil {
		return "", err
	}
	var v struct {
		CSRF string `json:"csrfToken"`
	}
	if response.StatusCode != 200 || json.Unmarshal(raw, &v) != nil || len(v.CSRF) < 32 || len(v.CSRF) > 128 {
		return "", errors.New("invalid upstream csrf")
	}
	for _, cookie := range response.Cookies() {
		if cookie.Name == "dysm_csrf" && cookie.Value == v.CSRF {
			return v.CSRF, nil
		}
	}
	return "", errors.New("missing upstream csrf cookie")
}

func (b *ysmBridge) sealConfirmation(v ysmConfirmation, session, path string) (string, error) {
	raw, _ := json.Marshal(v)
	nonce := make([]byte, b.seal.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b.seal.Seal(nonce, nonce, raw, []byte(session+"\n"+path))), nil
}

func (b *ysmBridge) openConfirmation(token, session, path string) (ysmConfirmation, error) {
	var v ysmConfirmation
	if len(token) > 4096 {
		return v, errors.New("oversized confirmation")
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) < b.seal.NonceSize() {
		return v, errors.New("invalid confirmation")
	}
	size := b.seal.NonceSize()
	plain, err := b.seal.Open(nil, raw[:size], raw[size:], []byte(session+"\n"+path))
	if err != nil {
		return v, err
	}
	if json.Unmarshal(plain, &v) != nil || v.Token == "" || v.CSRF == "" || time.Now().Unix() >= v.Expires {
		return v, errors.New("expired confirmation")
	}
	return v, nil
}

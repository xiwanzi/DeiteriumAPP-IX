package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

type desktopRuntime struct {
	nativeURL, nativeUser, nativePassword, bucket, publicURL, accessKey, secretKey string
	client                                                                         *http.Client
	objects                                                                        *s3.Client
	token                                                                          string
	tokenMu                                                                        sync.Mutex
	syncMu                                                                         sync.Mutex
}
type desktopIndexEntry struct {
	Label    string `json:"label"`
	Filename string `json:"filename"`
	Offset   int64  `json:"offset"`
	Length   int64  `json:"length"`
}

func (s *Server) initDesktopLauncher() {
	raw := os.Getenv("DEUTERIUM_LAUNCHER_MCPATCH_URL")
	if raw == "" {
		return
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" {
		return
	}
	d := &desktopRuntime{nativeURL: strings.TrimRight(raw, "/"), nativeUser: os.Getenv("DEUTERIUM_LAUNCHER_MCPATCH_USER"), nativePassword: os.Getenv("DEUTERIUM_LAUNCHER_MCPATCH_PASSWORD"),
		bucket: os.Getenv("DEUTERIUM_LAUNCHER_S3_BUCKET"), publicURL: strings.TrimRight(os.Getenv("DEUTERIUM_LAUNCHER_S3_PUBLIC"), "/"),
		accessKey: os.Getenv("DEUTERIUM_LAUNCHER_S3_ACCESS_KEY"), secretKey: os.Getenv("DEUTERIUM_LAUNCHER_S3_SECRET_KEY"), client: &http.Client{}}
	endpoint := os.Getenv("DEUTERIUM_LAUNCHER_S3_ENDPOINT")
	if !store.DesktopPublicURL(endpoint) || !store.DesktopPublicURL(d.publicURL) || d.bucket == "" || d.nativePassword == "" {
		return
	}
	d.objects = s3.New(s3.Options{Region: "us-east-1", BaseEndpoint: aws.String(endpoint), UsePathStyle: true, RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired, ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
		Credentials: credentials.NewStaticCredentialsProvider(d.accessKey, d.secretKey, "")})
	s.desktop = d
	if path := os.Getenv("DEUTERIUM_LAUNCHER_SEED_FILE"); path != "" {
		if raw, err := os.ReadFile(path); err == nil && len(raw) < 1024*1024 {
			var v store.DesktopLauncherDocument
			if json.Unmarshal(raw, &v) == nil {
				_ = s.Store.SeedDesktopLauncher(s.ctx, v)
			}
		}
	}
	go func() {
		// A service restart may have interrupted only the HTTP wait, while native McPatch kept uploading.
		v, err := s.Store.LatestDesktopSync(s.ctx)
		if err != nil || v == nil || v.State != "RUNNING" {
			return
		}
		ctx, cancel := context.WithTimeout(s.ctx, 45*time.Second)
		defer cancel()
		expected, err := d.nativePublic(ctx, "index.json", http.MethodGet)
		if err == nil {
			err = d.verifySynced(ctx, expected)
		}
		// Persist the outcome even when the verification deadline has expired.
		finishCtx, finishCancel := context.WithTimeout(context.WithoutCancel(s.ctx), 10*time.Second)
		defer finishCancel()
		if err == nil {
			_ = s.Store.FinishDesktopSync(finishCtx, v.ID, "SUCCEEDED", "已核对 OSS，同步内容完整。", expected)
		} else {
			_ = s.Store.FinishDesktopSync(finishCtx, v.ID, "FAILED", "上次同步等待中断。请在 McPatch 任务结束后重新同步，已上传文件会复用。", nil)
		}
	}()
}

func (d *desktopRuntime) redact(text string) string {
	d.tokenMu.Lock()
	token := d.token
	d.tokenMu.Unlock()
	for _, secret := range []string{d.nativePassword, d.accessKey, d.secretKey, token} {
		if secret != "" {
			text = strings.ReplaceAll(text, secret, "[已隐藏]")
		}
	}
	if len(text) > 2500 {
		text = text[:2500]
	}
	return text
}

func limitedBody(r *http.Response, limit int64) ([]byte, error) {
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > limit {
		return nil, errors.New("response too large")
	}
	return raw, nil
}

func (d *desktopRuntime) nativeToken(ctx context.Context, refresh bool) (string, error) {
	d.tokenMu.Lock()
	defer d.tokenMu.Unlock()
	if d.token != "" && !refresh {
		return d.token, nil
	}
	body, _ := json.Marshal(map[string]string{"username": d.nativeUser, "password": d.nativePassword})
	r, _ := http.NewRequestWithContext(ctx, http.MethodPost, d.nativeURL+"/api/user/login", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	response, err := d.client.Do(r)
	if err != nil {
		return "", errors.New("McPatch 暂时无法连接。")
	}
	raw, err := limitedBody(response, 65536)
	if err != nil {
		return "", err
	}
	var result struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &result) != nil || result.Code != 1 || result.Data.Token == "" {
		return "", errors.New("McPatch 服务端认证失败。")
	}
	d.token = result.Data.Token
	return d.token, nil
}

func (d *desktopRuntime) native(ctx context.Context, path string, input any, wait bool) ([]byte, error) {
	payload, _ := json.Marshal(input)
	for attempt := 0; attempt < 2; attempt++ {
		token, err := d.nativeToken(ctx, attempt > 0)
		if err != nil {
			return nil, err
		}
		request, _ := http.NewRequestWithContext(ctx, http.MethodPost, d.nativeURL+path, bytes.NewReader(payload))
		request.Header.Set("Token", token)
		request.Header.Set("Content-Type", "application/json")
		if wait {
			request.Header.Set("Wait", "1")
		}
		response, err := d.client.Do(request)
		if err != nil {
			return nil, errors.New("等待 McPatch 响应时连接中断。")
		}
		raw, err := limitedBody(response, 2*1024*1024)
		if err != nil {
			return nil, err
		}
		if response.StatusCode == http.StatusUnauthorized {
			continue
		}
		var envelope struct {
			Code *int   `json:"code"`
			Msg  string `json:"msg"`
		}
		if json.Unmarshal(raw, &envelope) == nil && envelope.Code != nil && *envelope.Code != 1 {
			if strings.Contains(strings.ToLower(envelope.Msg), "token") && attempt == 0 {
				continue
			}
			return nil, errors.New(d.redact(envelope.Msg))
		}
		if response.StatusCode >= 300 {
			return nil, fmt.Errorf("McPatch 任务失败（HTTP %d）。", response.StatusCode)
		}
		return raw, nil
	}
	return nil, errors.New("McPatch 会话失效，请重试。")
}

func (d *desktopRuntime) nativePublic(ctx context.Context, file, method string) ([]byte, error) {
	request, _ := http.NewRequestWithContext(ctx, method, d.nativeURL+"/public/"+url.PathEscape(file), nil)
	response, err := d.client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != 200 {
		response.Body.Close()
		return nil, errors.New("McPatch 尚未生成可用的更新包。")
	}
	if method == http.MethodHead {
		response.Body.Close()
		return []byte(fmt.Sprint(response.ContentLength)), nil
	}
	return limitedBody(response, 4*1024*1024)
}

func parseDesktopIndex(raw []byte) ([]desktopIndexEntry, error) {
	var entries []desktopIndexEntry
	if json.Unmarshal(raw, &entries) != nil || len(entries) == 0 || len(entries) > 10000 {
		return nil, errors.New("更新索引为空或无效。")
	}
	names := regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,139}\.tar$`)
	seen := map[string]bool{}
	for _, e := range entries {
		if !names.MatchString(e.Filename) || e.Label == "" || seen[e.Label] || e.Offset < 0 || e.Length <= 0 {
			return nil, errors.New("更新索引包含无效记录。")
		}
		seen[e.Label] = true
	}
	return entries, nil
}

func (d *desktopRuntime) verifySynced(ctx context.Context, expected []byte) error {
	entries, err := parseDesktopIndex(expected)
	if err != nil {
		return err
	}
	response, err := d.objects.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(d.bucket), Key: aws.String("index.json")})
	if err != nil {
		return errors.New("OSS 更新索引还未同步完成。")
	}
	remote, err := io.ReadAll(io.LimitReader(response.Body, 4*1024*1024+1))
	response.Body.Close()
	if err != nil || !bytes.Equal(bytes.TrimSpace(remote), bytes.TrimSpace(expected)) {
		return errors.New("OSS 索引与本次打包结果不一致，请重新同步。")
	}
	checked := map[string]bool{}
	for _, entry := range entries {
		if checked[entry.Filename] {
			continue
		}
		checked[entry.Filename] = true
		head, err := d.objects.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(d.bucket), Key: aws.String(entry.Filename)})
		if err != nil || aws.ToInt64(head.ContentLength) < entry.Offset+entry.Length {
			return errors.New("OSS 更新包缺失或尚未上传完整：" + entry.Filename)
		}
		local, err := d.nativePublic(ctx, entry.Filename, http.MethodHead)
		if err != nil || string(local) != fmt.Sprint(aws.ToInt64(head.ContentLength)) {
			return errors.New("更新包大小与 McPatch 不一致：" + entry.Filename)
		}
	}
	return nil
}

func (s *Server) registerDesktopLauncher(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/launcher/bootstrap", s.publicDesktopBootstrap)
	mux.HandleFunc("GET /api/v1/launcher/content", s.publicDesktopContent)
	mux.HandleFunc("GET /api/v1/launcher/updates/index.json", s.publicDesktopIndex)
	mux.HandleFunc("GET /api/v1/launcher/updates/{file}", s.publicDesktopPackage)
	mux.HandleFunc("GET /api/v1/admin/desktop-launcher", s.adminDesktopLauncher)
	mux.HandleFunc("PUT /api/v1/admin/desktop-launcher", s.saveDesktopLauncher)
	mux.HandleFunc("POST /api/v1/admin/desktop-launcher/sync", s.syncDesktopLauncher)
	mux.HandleFunc("POST /api/v1/admin/desktop-launcher/access", s.accessDesktopLauncher)
	mux.HandleFunc("POST /api/v1/admin/desktop-launcher/media", s.uploadDesktopMedia)
	mux.HandleFunc("POST /api/v1/admin/desktop-launcher/media/complete", s.completeDesktopMedia)
}

func (s *Server) publicDesktopBootstrap(w http.ResponseWriter, r *http.Request) {
	v, err := s.Store.DesktopLauncher(r.Context())
	if err != nil {
		failError(w, r, err)
		return
	}
	index, err := s.Store.DesktopPublishedIndex(r.Context())
	if err != nil || v.Published == nil || len(index) < 3 {
		failure(w, r, 503, "LAUNCHER_NOT_PUBLISHED", "客户端安装资源尚未发布。")
		return
	}
	v2Success(w, r, v.Published.Bootstrap)
}
func (s *Server) publicDesktopContent(w http.ResponseWriter, r *http.Request) {
	v, err := s.Store.DesktopLauncher(r.Context())
	if err != nil {
		failError(w, r, err)
		return
	}
	if v.Published == nil {
		failure(w, r, 404, "CONTENT_NOT_PUBLISHED", "启动器内容尚未发布。")
		return
	}
	v2Success(w, r, map[string]any{"version": v.PublishedVersion, "content": v.Published.Content})
}
func (s *Server) publicDesktopIndex(w http.ResponseWriter, r *http.Request) {
	index, err := s.Store.DesktopPublishedIndex(r.Context())
	if err != nil {
		failError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(index)
}
func (s *Server) publicDesktopPackage(w http.ResponseWriter, r *http.Request) {
	if s.desktop == nil {
		failure(w, r, 503, "LAUNCHER_UNAVAILABLE", "资源服务尚未连接。")
		return
	}
	index, err := s.Store.DesktopPublishedIndex(r.Context())
	if err != nil {
		failError(w, r, err)
		return
	}
	entries, err := parseDesktopIndex(index)
	if err != nil {
		failure(w, r, 404, "NOT_FOUND", "更新包尚未发布。")
		return
	}
	for _, entry := range entries {
		if entry.Filename == r.PathValue("file") {
			http.Redirect(w, r, s.desktop.publicURL+"/"+url.PathEscape(entry.Filename), http.StatusTemporaryRedirect)
			return
		}
	}
	failure(w, r, 404, "NOT_FOUND", "更新包不存在或尚未发布。")
}

func (s *Server) adminDesktopLauncher(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "platform.admin"); err != nil {
		failError(w, r, err)
		return
	}
	settings, err := s.Store.DesktopLauncher(r.Context())
	if err != nil {
		failError(w, r, err)
		return
	}
	latest, err := s.Store.LatestDesktopSync(r.Context())
	if err != nil {
		failError(w, r, err)
		return
	}
	state := map[string]any{"connected": s.desktop != nil, "url": s.Config.PublicOrigin + "/mcpatch/"}
	if s.desktop != nil {
		state["assetBaseUrl"] = s.desktop.publicURL + "/launcher/builtin/"
	}
	if s.desktop != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
		defer cancel()
		raw, err := s.desktop.nativePublic(ctx, "index.json", http.MethodGet)
		if err == nil {
			var v []desktopIndexEntry
			if json.Unmarshal(raw, &v) == nil && len(v) > 0 {
				state["versions"] = v
			} else {
				state["error"] = "McPatch 返回的版本索引无效，请检查服务连接。"
			}
		} else {
			state["error"] = s.desktop.redact(err.Error())
		}
	}
	v2Success(w, r, map[string]any{"settings": settings, "sync": latest, "mcpatch": state})
}

func (s *Server) saveDesktopLauncher(w http.ResponseWriter, r *http.Request) {
	actor, err := s.admin(r, "platform.admin")
	if err != nil {
		failError(w, r, err)
		return
	}
	var input struct {
		ClientRequestID string                        `json:"clientRequestId"`
		ExpectedVersion int64                         `json:"expectedVersion"`
		Document        store.DesktopLauncherDocument `json:"document"`
		Publish         bool                          `json:"publish"`
	}
	if err = socialBodyV2(w, r, &input, 1<<20); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	if input.Publish {
		var recipe struct {
			Baseline string `json:"baselineVersion"`
			URL      string `json:"mcpatchUrl"`
		}
		if json.Unmarshal(input.Document.Bootstrap, &recipe) != nil {
			failure(w, r, 400, "BASELINE_INVALID", "安装基线配置无效。")
			return
		}
		index, indexErr := s.Store.DesktopPublishedIndex(r.Context())
		entries, parseErr := parseDesktopIndex(index)
		found := false
		for _, entry := range entries {
			if entry.Label == recipe.Baseline {
				found = true
			}
		}
		if indexErr != nil || parseErr != nil || !found {
			failure(w, r, 409, "BASELINE_NOT_SYNCED", "请先将选定基线同步到 OSS，再发布启动器配置。")
			return
		}
	}
	result, err := s.Store.SaveDesktopLauncher(r.Context(), actor.ID, input.ClientRequestID, input.ExpectedVersion, input.Document, input.Publish)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	v2Success(w, r, result)
}

func (s *Server) syncDesktopLauncher(w http.ResponseWriter, r *http.Request) {
	actor, err := s.admin(r, "platform.admin")
	if err != nil {
		failError(w, r, err)
		return
	}
	if s.desktop == nil {
		failure(w, r, 503, "MCPATCH_UNAVAILABLE", "McPatch 尚未连接。")
		return
	}
	var input struct {
		ClientRequestID string `json:"clientRequestId"`
		NativeToken     string `json:"nativeToken,omitempty"`
	}
	if err = body(w, r, &input); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	if input.NativeToken != "" {
		if !regexp.MustCompile(`^[A-Za-z0-9]{16,128}$`).MatchString(input.NativeToken) {
			failure(w, r, 400, "MCPATCH_SESSION_INVALID", "McPatch 会话格式无效。")
			return
		}
		s.desktop.tokenMu.Lock()
		s.desktop.token = input.NativeToken
		s.desktop.tokenMu.Unlock()
	}
	raw, err := s.Store.BeginDesktopSync(r.Context(), actor.ID, input.ClientRequestID)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	var run store.DesktopLauncherSync
	if json.Unmarshal(raw, &run) == nil && s.desktop.syncMu.TryLock() {
		go s.runDesktopSync(run.ID)
	}
	v2Success(w, r, raw)
}

func (s *Server) runDesktopSync(id string) {
	defer s.desktop.syncMu.Unlock()
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Minute)
	defer cancel()
	latest, err := s.Store.LatestDesktopSync(ctx)
	if err != nil || latest == nil || latest.ID != id || latest.State != "RUNNING" {
		return
	}
	expected, err := s.desktop.nativePublic(ctx, "index.json", http.MethodGet)
	if err == nil {
		_, err = parseDesktopIndex(expected)
	}
	if err == nil {
		_, err = s.desktop.native(ctx, "/api/task/upload", map[string]any{}, true)
	}
	if err == nil {
		err = s.desktop.verifySynced(ctx, expected)
	}
	// A timed-out upload must still leave RUNNING so administrators can retry.
	finishCtx, finishCancel := context.WithTimeout(context.WithoutCancel(s.ctx), 10*time.Second)
	defer finishCancel()
	if err != nil {
		_ = s.Store.FinishDesktopSync(finishCtx, id, "FAILED", s.desktop.redact(err.Error()), nil)
		return
	}
	_ = s.Store.FinishDesktopSync(finishCtx, id, "SUCCEEDED", "最新更新已同步到 OSS，索引与文件完整性检查通过。", expected)
}

func (s *Server) accessDesktopLauncher(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "platform.admin"); err != nil {
		failError(w, r, err)
		return
	}
	if s.desktop == nil {
		failure(w, r, 503, "MCPATCH_UNAVAILABLE", "McPatch 尚未连接。")
		return
	}
	v2Success(w, r, map[string]string{"url": s.Config.PublicOrigin + "/mcpatch/", "username": s.desktop.nativeUser, "password": s.desktop.nativePassword})
}

func (s *Server) uploadDesktopMedia(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "platform.admin"); err != nil {
		failError(w, r, err)
		return
	}
	if s.desktop == nil {
		failure(w, r, 503, "S3_UNAVAILABLE", "资源站尚未配置。")
		return
	}
	var input struct {
		Type   string `json:"contentType"`
		Size   int64  `json:"size"`
		SHA256 string `json:"sha256"`
	}
	if err := body(w, r, &input); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	extension := map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/webp": ".webp", "video/mp4": ".mp4"}[input.Type]
	digest, err := hex.DecodeString(input.SHA256)
	if extension == "" || input.Size < 1 || input.Size > 128*1024*1024 || err != nil || len(digest) != 32 {
		failure(w, r, 400, "MEDIA_INVALID", "请选择 128 MB 以内的 PNG、JPEG、WebP 或 MP4。")
		return
	}
	key := "launcher/media/" + strings.ToLower(input.SHA256) + extension
	signed, err := s3.NewPresignClient(s.desktop.objects).PresignPutObject(r.Context(), &s3.PutObjectInput{Bucket: aws.String(s.desktop.bucket), Key: aws.String(key), ContentType: aws.String(input.Type), ContentLength: aws.Int64(input.Size)}, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		failure(w, r, 503, "UPLOAD_UNAVAILABLE", "无法创建上传任务。")
		return
	}
	v2Success(w, r, map[string]any{"key": key, "uploadUrl": signed.URL, "headers": signed.SignedHeader, "url": s.desktop.publicURL + "/" + key})
}

func (s *Server) completeDesktopMedia(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "platform.admin"); err != nil {
		failError(w, r, err)
		return
	}
	if s.desktop == nil {
		failure(w, r, 503, "S3_UNAVAILABLE", "资源站尚未配置。")
		return
	}
	var input struct {
		Key string `json:"key"`
	}
	if err := body(w, r, &input); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	pattern := regexp.MustCompile(`^launcher/media/([a-f0-9]{64})\.(png|jpg|webp|mp4)$`)
	match := pattern.FindStringSubmatch(input.Key)
	if match == nil {
		failure(w, r, 400, "MEDIA_INVALID", "上传对象无效。")
		return
	}
	object, err := s.desktop.objects.GetObject(r.Context(), &s3.GetObjectInput{Bucket: aws.String(s.desktop.bucket), Key: aws.String(input.Key)})
	if err != nil {
		failure(w, r, 409, "MEDIA_NOT_READY", "上传尚未完成。")
		return
	}
	defer object.Body.Close()
	digest := sha256.New()
	size, err := io.Copy(digest, io.LimitReader(object.Body, 128*1024*1024+1))
	if err != nil || size > 128*1024*1024 || hex.EncodeToString(digest.Sum(nil)) != match[1] {
		failure(w, r, 409, "MEDIA_HASH_MISMATCH", "上传文件校验失败，请重试。")
		return
	}
	v2Success(w, r, map[string]any{"url": s.desktop.publicURL + "/" + input.Key, "size": size, "sha256": match[1]})
}

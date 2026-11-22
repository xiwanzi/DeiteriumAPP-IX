package httpapi

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var desktopApplicationPublicKey, _ = hex.DecodeString("f3b2e0109aaae39d76f8c8e0c9ea37143f5a11b40305101d88fc4c168dbbcc9a")

const desktopApplicationLimit int64 = 1024 * 1024 * 1024
const desktopApplicationManifestLimit int64 = 8 * 1024 * 1024

type desktopApplicationArtifact struct {
	URL    string `json:"url"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type desktopApplicationEnvelope struct {
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

type desktopApplicationRelease struct {
	SchemaVersion   int                         `json:"schemaVersion"`
	Product         string                      `json:"product"`
	Platform        string                      `json:"platform"`
	Version         string                      `json:"version"`
	VersionCode     int64                       `json:"versionCode"`
	UpdaterProtocol int                         `json:"updaterProtocol"`
	Notes           string                      `json:"notes"`
	PublishedAt     string                      `json:"publishedAt"`
	Package         desktopApplicationArtifact  `json:"package"`
	Manifest        *desktopApplicationArtifact `json:"manifest,omitempty"`
}

func strictApplicationJSON(raw []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return errors.New("unexpected JSON suffix")
	}
	return nil
}

func applicationFields(raw []byte, names ...string) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, errors.New("object required")
	}
	allowed := map[string]bool{}
	for _, name := range names {
		allowed[name] = true
	}
	fields := map[string]json.RawMessage{}
	for decoder.More() {
		token, err = decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok || !allowed[key] || fields[key] != nil {
			return nil, errors.New("unexpected or duplicate field")
		}
		var value json.RawMessage
		if err = decoder.Decode(&value); err != nil {
			return nil, err
		}
		fields[key] = value
	}
	if _, err = decoder.Token(); err != nil || len(fields) != len(names) {
		return nil, errors.New("missing field")
	}
	return fields, nil
}

func verifyDesktopApplication(raw json.RawMessage, publicKey ed25519.PublicKey) (desktopApplicationRelease, error) {
	var envelope desktopApplicationEnvelope
	var value desktopApplicationRelease
	invalid := errors.New("invalid signed launcher release")
	if _, err := applicationFields(raw, "payload", "signature"); err != nil {
		return value, invalid
	}
	if len(raw) > 64000 || strictApplicationJSON(raw, &envelope) != nil || len(envelope.Payload) > 48000 || len(envelope.Signature) != 88 {
		return value, invalid
	}
	payload, err := base64.StdEncoding.Strict().DecodeString(envelope.Payload)
	signature, signErr := base64.StdEncoding.Strict().DecodeString(envelope.Signature)
	if err != nil || signErr != nil || len(publicKey) != ed25519.PublicKeySize || len(payload) > 36000 || base64.StdEncoding.EncodeToString(payload) != envelope.Payload || base64.StdEncoding.EncodeToString(signature) != envelope.Signature || !ed25519.Verify(publicKey, payload, signature) || strictApplicationJSON(payload, &value) != nil {
		return value, invalid
	}
	names := []string{"schemaVersion", "product", "platform", "version", "versionCode", "updaterProtocol", "notes", "publishedAt", "package"}
	if value.UpdaterProtocol == 2 {
		names = append(names, "manifest")
	}
	fields, err := applicationFields(payload, names...)
	if err != nil {
		return value, invalid
	}
	if _, err = applicationFields(fields["package"], "url", "size", "sha256"); err != nil {
		return value, invalid
	}
	if value.SchemaVersion != 1 || value.Product != "DLauncher" || value.Platform != "windows-x64" || (value.UpdaterProtocol != 1 && value.UpdaterProtocol != 2) || len([]rune(value.Notes)) > 8000 {
		return value, invalid
	}
	if !regexp.MustCompile(`^(0|[1-9][0-9]{0,2})\.(0|[1-9][0-9]{0,2})\.(0|[1-9][0-9]{0,2})$`).MatchString(value.Version) {
		return value, invalid
	}
	parts := strings.Split(value.Version, ".")
	major, _ := strconv.ParseInt(parts[0], 10, 64)
	minor, _ := strconv.ParseInt(parts[1], 10, 64)
	patch, _ := strconv.ParseInt(parts[2], 10, 64)
	if value.VersionCode != major*1000000+minor*1000+patch || value.VersionCode < 1 {
		return value, invalid
	}
	if _, err = time.Parse("2006-01-02T15:04:05Z", value.PublishedAt); err != nil {
		return value, invalid
	}
	u, err := url.Parse(value.Package.URL)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || value.Package.Size < 1 || value.Package.Size > desktopApplicationLimit || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(value.Package.SHA256) {
		return value, invalid
	}
	if !strings.HasSuffix(u.Path, "/"+desktopApplicationObjectKey(value)) {
		return value, invalid
	}
	if value.UpdaterProtocol == 2 {
		if _, err = applicationFields(fields["manifest"], "url", "size", "sha256"); err != nil || value.Manifest == nil {
			return value, invalid
		}
		m := value.Manifest
		base := strings.TrimSuffix(value.Package.URL, desktopApplicationObjectKey(value))
		if m.Size < 1 || m.Size > desktopApplicationManifestLimit || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(m.SHA256) || m.URL != base+desktopApplicationManifestKey(m.SHA256) {
			return value, invalid
		}
	}
	return value, nil
}

func desktopApplicationObjectKey(value desktopApplicationRelease) string {
	return "launcher/application/windows-x64/" + value.Package.SHA256 + ".zip"
}

func desktopApplicationManifestKey(digest string) string {
	return "launcher/application/windows-x64/manifests/" + digest + ".json"
}

func desktopApplicationFileKey(digest string) string {
	return "launcher/application/windows-x64/files/" + digest
}

func desktopApplicationEnvelopes(raw json.RawMessage) (legacy, incremental json.RawMessage, err error) {
	var marker struct {
		SchemaVersion int `json:"schemaVersion"`
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil, nil
	}
	if err = json.Unmarshal(raw, &marker); err != nil {
		return nil, nil, err
	}
	if marker.SchemaVersion != 2 {
		return raw, nil, nil
	}
	fields, err := applicationFields(raw, "schemaVersion", "legacy", "incremental")
	if err != nil {
		return nil, nil, err
	}
	return fields["legacy"], fields["incremental"], nil
}

func verifyDesktopPublication(raw json.RawMessage, publicKey ed25519.PublicKey) (desktopApplicationRelease, error) {
	invalid := errors.New("invalid launcher publication")
	legacy, incremental, err := desktopApplicationEnvelopes(raw)
	if err != nil || len(raw) > 128000 {
		return desktopApplicationRelease{}, invalid
	}
	previous, err := verifyDesktopApplication(legacy, publicKey)
	if err != nil || previous.UpdaterProtocol != 1 {
		return previous, invalid
	}
	if incremental == nil {
		return previous, nil
	}
	current, err := verifyDesktopApplication(incremental, publicKey)
	if err != nil || current.UpdaterProtocol != 2 || previous.Version != current.Version || previous.VersionCode != current.VersionCode || previous.PublishedAt != current.PublishedAt || previous.Notes != current.Notes || previous.Package != current.Package {
		return current, invalid
	}
	return current, nil
}

func (s *Server) registerDesktopApplication(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/launcher/application", s.publicDesktopApplication)
	mux.HandleFunc("GET /api/v1/admin/desktop-launcher/application", s.adminDesktopApplication)
	mux.HandleFunc("POST /api/v1/admin/desktop-launcher/application/upload", s.uploadDesktopApplication)
	mux.HandleFunc("POST /api/v1/admin/desktop-launcher/application/publish", s.publishDesktopApplication)
}

func (s *Server) publicDesktopApplication(w http.ResponseWriter, r *http.Request) {
	value, err := s.Store.DesktopApplication(r.Context())
	if err != nil {
		failError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	legacy, incremental, err := desktopApplicationEnvelopes(value.Release)
	if err != nil {
		failError(w, r, err)
		return
	}
	if r.URL.Query().Get("updaterProtocol") == "2" && incremental != nil {
		legacy = incremental
	}
	v2Success(w, r, map[string]any{"release": legacy})
}

func (s *Server) adminDesktopApplication(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "platform.admin"); err != nil {
		failError(w, r, err)
		return
	}
	value, err := s.Store.DesktopApplication(r.Context())
	if err != nil {
		failError(w, r, err)
		return
	}
	legacy, incremental, err := desktopApplicationEnvelopes(value.Release)
	if err != nil {
		failError(w, r, err)
		return
	}
	v2Success(w, r, map[string]any{"application": map[string]any{"revision": value.Revision, "version": value.Version, "versionCode": value.VersionCode, "publishedAt": value.PublishedAt, "release": legacy, "incrementalRelease": incremental}, "uploadReady": s.desktop != nil})
}

func (s *Server) uploadDesktopApplication(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "platform.admin"); err != nil {
		failError(w, r, err)
		return
	}
	var input struct {
		Release json.RawMessage `json:"release"`
	}
	if err := socialBodyV2(w, r, &input, 192000); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	value, err := verifyDesktopPublication(input.Release, desktopApplicationPublicKey)
	if err != nil {
		failure(w, r, 400, "APPLICATION_SIGNATURE_INVALID", "版本签名或更新清单无效，请使用构建生成的版本文件。")
		return
	}
	if s.desktop == nil || value.Package.URL != strings.TrimRight(s.desktop.publicURL, "/")+"/"+desktopApplicationObjectKey(value) {
		failure(w, r, 409, "APPLICATION_STORAGE_MISMATCH", "更新包的发布地址与当前 OSS 配置不一致。")
		return
	}
	signed, err := s3.NewPresignClient(s.desktop.objects).PresignPutObject(r.Context(), &s3.PutObjectInput{Bucket: aws.String(s.desktop.bucket), Key: aws.String(desktopApplicationObjectKey(value)), ContentType: aws.String("application/zip"), ContentLength: aws.Int64(value.Package.Size)}, s3.WithPresignExpires(30*time.Minute))
	if err != nil {
		failure(w, r, 503, "UPLOAD_UNAVAILABLE", "无法创建上传任务，请重试。")
		return
	}
	v2Success(w, r, map[string]any{"uploadUrl": signed.URL, "headers": signed.SignedHeader, "version": value.Version})
}

func (s *Server) publishDesktopApplication(w http.ResponseWriter, r *http.Request) {
	actor, err := s.admin(r, "platform.admin")
	if err != nil {
		failError(w, r, err)
		return
	}
	var input struct {
		ClientRequestID string          `json:"clientRequestId"`
		Expected        int64           `json:"expectedRevision"`
		Release         json.RawMessage `json:"release"`
	}
	if err = socialBodyV2(w, r, &input, 192000); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	value, err := verifyDesktopPublication(input.Release, desktopApplicationPublicKey)
	if err != nil {
		failure(w, r, 400, "APPLICATION_SIGNATURE_INVALID", "版本签名校验失败。")
		return
	}
	if s.desktop == nil || value.Package.URL != strings.TrimRight(s.desktop.publicURL, "/")+"/"+desktopApplicationObjectKey(value) {
		failure(w, r, 409, "APPLICATION_STORAGE_MISMATCH", "发布地址与当前 OSS 配置不一致。")
		return
	}
	if !s.desktop.applicationMu.TryLock() {
		w.Header().Set("Retry-After", "5")
		failure(w, r, 503, "APPLICATION_PUBLISH_BUSY", "另一个启动器版本正在准备发布，请稍后重试。")
		return
	}
	defer s.desktop.applicationMu.Unlock()
	publish := func() {
		result, err := s.Store.PublishDesktopApplication(r.Context(), actor.ID, input.ClientRequestID, input.Expected, value.Version, value.VersionCode, input.Release)
		if err != nil {
			socialFailureV2(w, r, err)
			return
		}
		v2Success(w, r, result)
	}
	current, err := s.Store.DesktopApplication(r.Context())
	if err != nil {
		failError(w, r, err)
		return
	}
	if current.Revision != input.Expected || current.VersionCode >= value.VersionCode {
		publish() // Resolve exact idempotent replays or reject stale/conflicting writes.
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(11 * time.Minute))
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	object, err := s.desktop.objects.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.desktop.bucket), Key: aws.String(desktopApplicationObjectKey(value))})
	if err != nil {
		failure(w, r, 409, "APPLICATION_NOT_UPLOADED", "尚未找到完整更新包，请完成上传后重试。")
		return
	}
	defer object.Body.Close()
	temporary, err := os.CreateTemp("", "deuterium-launcher-package-*.zip")
	if err != nil {
		failure(w, r, 503, "APPLICATION_TEMPORARY_STORAGE", "暂时无法准备更新包校验空间。")
		return
	}
	defer func() { temporary.Close(); os.Remove(temporary.Name()) }()
	digest := sha256.New()
	size, err := io.Copy(io.MultiWriter(digest, temporary), io.LimitReader(object.Body, value.Package.Size+1))
	if err != nil || size != value.Package.Size || hex.EncodeToString(digest.Sum(nil)) != value.Package.SHA256 {
		failure(w, r, 409, "APPLICATION_HASH_MISMATCH", "更新包完整性校验失败，当前发布版本保持不变。")
		return
	}
	if value.UpdaterProtocol == 2 {
		if err := s.desktop.prepareApplicationFiles(ctx, value, temporary, size); err != nil {
			failure(w, r, 409, "APPLICATION_INCREMENTAL_NOT_READY", "增量文件准备或校验失败，当前发布版本保持不变，请重试。")
			return
		}
	}
	publish()
}

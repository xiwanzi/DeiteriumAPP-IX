//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/imagereview"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/testdb"
)

type reviewHTTPObjects struct{ deleted int }

func (o *reviewHTTPObjects) Inspect(context.Context, imagereview.Candidate) (imagereview.ObjectInfo, error) {
	return imagereview.ObjectInfo{}, nil
}
func (o *reviewHTTPObjects) URL(context.Context, string) (string, error) {
	return "https://oss.example/private-evidence?signature=hidden", nil
}
func (o *reviewHTTPObjects) EvidenceKey(id string) string                                { return "evidence/" + id }
func (o *reviewHTTPObjects) Retain(context.Context, string, string, string, int64) error { return nil }
func (o *reviewHTTPObjects) DeleteEvidence(context.Context, string) error                { o.deleted++; return nil }
func TestImageReviewWebPermissionsCSRFAndEvidenceAccess(t *testing.T) {
	ctx := context.Background()
	source := testdb.New(t)
	audit := testdb.New(t)
	repo := &imagereview.Repository{DB: audit.DB}
	if e := repo.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	tokens := map[string]string{}
	for n, kind := range []string{"admin", "support", "player"} {
		token := identity.Secret()
		tokens[kind] = token
		_, e := source.DB.Exec(`INSERT INTO identities(id,player_ref,server_uuid,game_id,qq,password_hash,status,created_at,updated_at,legacy_fingerprint) VALUES(?,?,?,?,?,?,'active',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),?)`, kind, "player_"+kind, fmt.Sprintf("11111111-1111-1111-1111-%012d", n), kind, fmt.Sprintf("100%d", n), "unused", strings.Repeat("0", 64))
		if e != nil {
			t.Fatal(e)
		}
		if e = source.CreateSession(ctx, store.User{ID: kind, PasswordHash: "unused"}, store.Digest([]byte(token)), "web", "csrf-test", time.Now().Add(time.Hour), ""); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := source.DB.Exec("INSERT INTO identity_permissions VALUES('admin','platform.admin'),('support','support.manage')"); e != nil {
		t.Fatal(e)
	}
	id := imagereview.ReviewID("ASSET", "upload_http")
	c := imagereview.Candidate{ID: id, SourceID: "upload_http", SourceKind: "ASSET", SourceKey: "original/keep.png", GameID: "Uploader", SizeBytes: 100, UploadedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}
	if e := repo.Discover(ctx, c); e != nil {
		t.Fatal(e)
	}
	if e := repo.Scored(ctx, imagereview.Item{Candidate: c}, imagereview.Result{Scores: imagereview.Scores{Promotion: 80}, Summary: "联系信息", Reasons: []string{"可见联系方式"}}, "evidence/"+id, `"etag"`); e != nil {
		t.Fatal(e)
	}
	if e := repo.Finish(ctx, id, "FLAGGED"); e != nil {
		t.Fatal(e)
	}
	objects := &reviewHTTPObjects{}
	service := &imagereview.Service{Repo: repo, Source: &imagereview.Source{DB: source.DB}, Objects: objects}
	handler := ImageReviewHandler(source, config.Config{Development: true, PublicOrigin: "https://admin.example"}, service)
	call := func(user, method, path, origin, csrf string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, nil)
		if user != "" {
			r.AddCookie(&http.Cookie{Name: "deuterium_dev_session", Value: tokens[user]})
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if csrf != "" {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s %s => %d want %d: %s", user, method, path, w.Code, want, w.Body.String())
		}
		return w
	}
	path := "/api/v1/admin/image-review/" + id
	for _, p := range []string{"/api/v1/admin/image-review", path, path + "/image"} {
		call("", "GET", p, "", "", 401)
		call("player", "GET", p, "", "", 403)
		call("support", "GET", p, "", "", 403)
	}
	listing := call("admin", "GET", "/api/v1/admin/image-review", "", "", 200)
	for _, secret := range []string{"sourceKey", "evidenceKey", "original/", "signature="} {
		if strings.Contains(listing.Body.String(), secret) {
			t.Fatal("list leaked storage details", secret)
		}
	}
	image := call("admin", "GET", path+"/image", "", "", 307)
	if image.Header().Get("Cache-Control") != "no-store" || image.Header().Get("Location") == "" {
		t.Fatal("image not private redirect")
	}
	call("admin", "DELETE", path, "https://evil.example", "csrf-test", 403)
	call("admin", "DELETE", path, "https://admin.example", "wrong", 403)
	call("support", "DELETE", path, "https://admin.example", "csrf-test", 403)
	if objects.deleted != 0 {
		t.Fatal("unauthorized deletion")
	}
	response := call("admin", "DELETE", path, "https://admin.example", "csrf-test", 200)
	var body map[string]any
	json.Unmarshal(response.Body.Bytes(), &body)
	if body["data"].(map[string]any)["state"] != "DELETED" {
		t.Fatal(body)
	}
	call("admin", "GET", path+"/image", "", "", 404)
	call("admin", "DELETE", path, "https://admin.example", "csrf-test", 200)
	if objects.deleted != 1 {
		t.Fatal("delete replay touched object twice")
	}
}

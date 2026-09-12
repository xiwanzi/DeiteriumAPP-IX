//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func desktopDocument(t *testing.T) store.DesktopLauncherDocument {
	t.Helper()
	artifact := map[string]any{"url": "https://launcher.example/artifact", "size": 100, "sha256": strings.Repeat("a", 64)}
	bootstrap, _ := json.Marshal(map[string]any{"minecraftVersion": "1.21.1", "loaderVersion": "21.1.248", "instanceVersion": "1.21.1-NeoForge_21.1.248", "baselineVersion": "baseline", "mcpatchUrl": "https://launcher.example/updates/", "java": artifact, "installer": artifact})
	entries := []any{}
	for _, key := range []string{"arknights", "endfield", "popucom"} {
		entries = append(entries, map[string]any{"key": key, "name": key, "icon": "hypergryph/icon.png", "background": "hypergryph/bg.png", "gallery": "hypergryph/gallery.webp", "cover": "hypergryph/cover.webp", "accent": "#fdfc00", "hover": "#ffff00", "pressed": "#dddd00", "tabs": []string{"公告"}, "sidebars": []any{}, "banners": []any{}, "news": []any{map[string]any{"title": "完整公告保存验证", "tab": "公告", "date": "09/12", "body": strings.Repeat("正文", 3000), "images": []any{}}}})
	}
	content, _ := json.Marshal(map[string]any{"entries": entries})
	return store.DesktopLauncherDocument{Bootstrap: bootstrap, Content: content}
}

func TestDesktopLauncherPermissionsDraftPublishAndReplay(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	document := desktopDocument(t)
	if err := f.store.SeedDesktopLauncher(ctx, document); err != nil {
		t.Fatal(err)
	}
	endpoint := "/api/v1/admin/desktop-launcher"
	f.request(t, "", "GET", endpoint, nil, 401)
	f.request(t, "Bob", "GET", endpoint, nil, 403)
	f.request(t, "Bob", "POST", endpoint+"/access", map[string]any{}, 403)
	f.request(t, "Bob", "POST", endpoint+"/sync", map[string]any{"clientRequestId": "denied"}, 403)
	if err := f.store.Grant(ctx, f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	f.request(t, "Alice", "GET", endpoint, nil, 200)
	input := map[string]any{"clientRequestId": "draft", "expectedVersion": 1, "document": document, "publish": false}
	first := socialDataV2(f.request(t, "Alice", "PUT", endpoint, input, 200))
	again := socialDataV2(f.request(t, "Alice", "PUT", endpoint, input, 200))
	if first["version"] != float64(2) || again["version"] != first["version"] {
		t.Fatal("draft replay changed version")
	}
	f.request(t, "", "GET", "/api/v1/launcher/content", nil, 404)
	input = map[string]any{"clientRequestId": "publish-before-sync", "expectedVersion": 2, "document": document, "publish": true}
	f.request(t, "Alice", "PUT", endpoint, input, 409)
	raw, err := f.store.BeginDesktopSync(ctx, f.users["Alice"].ID, "seed-sync")
	if err != nil {
		t.Fatal(err)
	}
	var run store.DesktopLauncherSync
	json.Unmarshal(raw, &run)
	index := []byte(`[{"label":"baseline","filename":"baseline.tar","offset":512,"length":128,"hash":"fixture"}]`)
	if err = f.store.FinishDesktopSync(ctx, run.ID, "SUCCEEDED", "verified fixture", index); err != nil {
		t.Fatal(err)
	}
	input["clientRequestId"] = "publish"
	f.request(t, "Alice", "PUT", endpoint, input, 200)
	f.request(t, "Alice", "PUT", endpoint, input, 200)
	f.request(t, "", "GET", "/api/v1/launcher/content", nil, 200)
	f.request(t, "", "GET", "/api/v1/launcher/bootstrap", nil, 200)
	input["clientRequestId"] = "stale"
	f.request(t, "Alice", "PUT", endpoint, input, 409)
	var count int
	if err = f.store.DB.QueryRow("SELECT COUNT(*) FROM audit_events_next WHERE action='desktop-launcher.settings'").Scan(&count); err != nil || count != 2 {
		t.Fatalf("duplicate audit %d %v", count, err)
	}
	persisted, err := f.store.DesktopLauncher(ctx)
	if err != nil || persisted.Published == nil || persisted.PublishedVersion != 3 {
		t.Fatal("publication not durable", err)
	}
}

func TestDesktopLauncherWebWritesRequireCSRF(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	if err := f.store.Grant(ctx, f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	token, csrf := identity.Secret(), identity.Secret()
	if err := f.store.CreateSession(ctx, f.users["Alice"], store.Digest([]byte(token)), "web", csrf, time.Now().Add(time.Hour), ""); err != nil {
		t.Fatal(err)
	}
	input := map[string]any{"clientRequestId": "csrf-draft", "expectedVersion": 1, "document": desktopDocument(t), "publish": false}
	encoded, _ := json.Marshal(input)
	for _, attempt := range []struct {
		origin, csrf string
		status       int
	}{{f.http.URL, "", 403}, {"https://other.example", csrf, 403}, {f.http.URL, csrf, 200}} {
		request, _ := http.NewRequest("PUT", f.http.URL+"/api/v1/admin/desktop-launcher", bytes.NewReader(encoded))
		request.AddCookie(&http.Cookie{Name: "deuterium_dev_session", Value: token})
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", attempt.origin)
		request.Header.Set("X-CSRF-Token", attempt.csrf)
		response, err := f.http.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != attempt.status {
			t.Fatalf("CSRF status %d expected %d", response.StatusCode, attempt.status)
		}
	}
}

func TestDesktopLauncherNativeFailureCannotPublishMissingPackages(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	if err := f.store.Grant(ctx, f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	index := []byte(`[{"label":"baseline","filename":"baseline.tar","offset":512,"length":128,"hash":"fixture"}]`)
	var ready atomic.Bool
	var uploads atomic.Int32
	native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/login":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"code":1,"data":{"token":"abcdefghijklmnopqrstuvwx"}}`))
		case "/public/index.json":
			w.Write(index)
		case "/public/baseline.tar":
			w.Header().Set("Content-Length", "1024")
			w.WriteHeader(200)
		case "/api/task/upload":
			uploads.Add(1)
			w.Write([]byte("native task response"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer native.Close()
	objects := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/dlt/index.json" {
			w.Write(index)
			return
		}
		if r.URL.Path == "/dlt/baseline.tar" && ready.Load() {
			w.Header().Set("Content-Length", "1024")
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(404)
		w.Write([]byte(`<Error><Code>NoSuchKey</Code></Error>`))
	}))
	defer objects.Close()
	f.app.desktop = &desktopRuntime{nativeURL: native.URL, nativeUser: "admin", nativePassword: "fixture", bucket: "dlt", publicURL: "https://files.example", client: native.Client(),
		objects: s3.New(s3.Options{Region: "us-east-1", BaseEndpoint: aws.String(objects.URL), UsePathStyle: true, Credentials: credentials.NewStaticCredentialsProvider("fixture", "fixture", "")})}
	wait := func(state string) {
		t.Helper()
		deadline := time.Now().Add(8 * time.Second)
		for time.Now().Before(deadline) {
			v, err := f.store.LatestDesktopSync(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if v != nil && v.State == state {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatal("sync did not reach", state)
	}
	endpoint := "/api/v1/admin/desktop-launcher/sync"
	f.request(t, "Alice", "POST", endpoint, map[string]any{"clientRequestId": "failed-upload"}, 200)
	wait("FAILED")
	published, err := f.store.DesktopPublishedIndex(ctx)
	if err != nil || string(published) != "[]" {
		t.Fatal("failed native upload was published")
	}
	ready.Store(true)
	input := map[string]any{"clientRequestId": "retry-upload"}
	f.request(t, "Alice", "POST", endpoint, input, 200)
	wait("SUCCEEDED")
	f.request(t, "Alice", "POST", endpoint, input, 200)
	time.Sleep(50 * time.Millisecond)
	if uploads.Load() != 2 {
		t.Fatalf("replayed request started upload again: %d", uploads.Load())
	}
	published, err = f.store.DesktopPublishedIndex(ctx)
	if err != nil || !bytes.Equal(published, index) {
		t.Fatal("verified index was not published")
	}
}

func TestDesktopLauncherConcurrentDraftHasOneWinner(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	if err := f.store.Grant(ctx, f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	document := desktopDocument(t)
	var wait sync.WaitGroup
	statuses := make(chan int, 4)
	for i := 0; i < 4; i++ {
		wait.Add(1)
		go func(n int) {
			defer wait.Done()
			code, _, _ := f.call("Alice", "PUT", "/api/v1/admin/desktop-launcher", map[string]any{"clientRequestId": string(rune('a' + n)), "expectedVersion": 1, "document": document, "publish": false})
			statuses <- code
		}(i)
	}
	wait.Wait()
	close(statuses)
	won := 0
	for code := range statuses {
		if code == 200 {
			won++
		} else if code != 409 {
			t.Fatalf("unexpected concurrent status %d", code)
		}
	}
	if won != 1 {
		t.Fatal("concurrent edits did not have one winner", won)
	}
}

//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestDesktopApplicationPermissionsPublicationAndReplay(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	endpoint := "/api/v1/admin/desktop-launcher/application"
	f.request(t, "", "GET", endpoint, nil, 401)
	f.request(t, "Bob", "GET", endpoint, nil, 403)
	f.request(t, "Bob", "POST", endpoint+"/upload", map[string]any{}, 403)
	f.request(t, "Bob", "POST", endpoint+"/publish", map[string]any{}, 403)
	if err := f.store.Grant(ctx, f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	f.request(t, "Alice", "GET", endpoint, nil, 200)
	public := socialDataV2(f.request(t, "", "GET", "/api/v1/launcher/application", nil, 200))
	if public["release"] != nil {
		t.Fatal("unpublished updater release must be null")
	}
	raw, _ := signedDesktopApplication(t, nil)
	f.request(t, "Alice", "POST", endpoint+"/upload", map[string]any{"release": json.RawMessage(raw)}, 400)
	f.request(t, "Alice", "POST", endpoint+"/publish", map[string]any{"clientRequestId": "invalid-signature", "expectedRevision": 0, "release": json.RawMessage(raw)}, 400)
	// Store transitions are tested with a non-production signed fixture; the HTTP
	// signature boundary above correctly refuses this fixture's independent key.
	first, err := f.store.PublishDesktopApplication(ctx, f.users["Alice"].ID, "release-one", 0, "0.6.0", 6000, raw)
	if err != nil {
		t.Fatal(err)
	}
	again, err := f.store.PublishDesktopApplication(ctx, f.users["Alice"].ID, "release-one", 0, "0.6.0", 6000, raw)
	if err != nil || string(first) != string(again) {
		t.Fatal("publication retry was not idempotent", err)
	}
	if _, err = f.store.PublishDesktopApplication(ctx, f.users["Alice"].ID, "stale", 0, "0.6.1", 6001, raw); !errors.Is(err, store.ErrSocialVersion) {
		t.Fatal("stale revision was not rejected", err)
	}
	if _, err = f.store.PublishDesktopApplication(ctx, f.users["Alice"].ID, "downgrade", 1, "0.5.1", 5001, raw); !errors.Is(err, store.ErrConflict) {
		t.Fatal("release downgrade was not rejected", err)
	}
	state, err := f.store.DesktopApplication(ctx)
	if err != nil || state.Revision != 1 || state.VersionCode != 6000 || !strings.Contains(string(state.Release), "signature") {
		t.Fatal("publication was not durable", err)
	}
	var count int
	if err = f.store.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_events_next WHERE action='desktop-launcher.application'").Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate publication audit", count, err)
	}
}

func TestDesktopApplicationRealArtifactPublication(t *testing.T) {
	manifestPath, packagePath := os.Getenv("DEUTERIUM_APPLICATION_QA_MANIFEST"), os.Getenv("DEUTERIUM_APPLICATION_QA_ZIP")
	if manifestPath == "" || packagePath == "" {
		t.Skip("opt-in verification of the actual signed application artifact")
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	release, err := verifyDesktopApplication(raw, desktopApplicationPublicKey)
	if err != nil {
		t.Fatal(err)
	}
	f := newSocialFixtureV2(t)
	if err = f.store.Grant(context.Background(), f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	var corrupt atomic.Bool
	corrupt.Store(true)
	objects := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/fixture/"+desktopApplicationObjectKey(release) {
			t.Errorf("unexpected object request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)
			return
		}
		if corrupt.Load() {
			w.Write([]byte("incomplete upload fixture"))
			return
		}
		file, err := os.Open(packagePath)
		if err != nil {
			t.Error(err)
			w.WriteHeader(500)
			return
		}
		defer file.Close()
		io.Copy(w, file)
	}))
	defer objects.Close()
	f.app.desktop = &desktopRuntime{bucket: "fixture", publicURL: strings.TrimSuffix(release.Package.URL, "/"+desktopApplicationObjectKey(release)), objects: s3.New(s3.Options{Region: "us-east-1", BaseEndpoint: aws.String(objects.URL), UsePathStyle: true, Credentials: credentials.NewStaticCredentialsProvider("fixture", "fixture", "")})}
	endpoint := "/api/v1/admin/desktop-launcher/application"
	prepared := socialDataV2(f.request(t, "Alice", "POST", endpoint+"/upload", map[string]any{"release": json.RawMessage(raw)}, 200))
	if prepared["version"] != release.Version {
		t.Fatal("prepare returned wrong version")
	}
	input := map[string]any{"clientRequestId": "real-artifact-publication", "expectedRevision": 0, "release": json.RawMessage(raw)}
	f.request(t, "Alice", "POST", endpoint+"/publish", input, 409)
	before, _ := f.store.DesktopApplication(context.Background())
	if before.Revision != 0 {
		t.Fatal("incomplete upload changed publication")
	}
	corrupt.Store(false)
	f.request(t, "Alice", "POST", endpoint+"/publish", input, 200)
	f.request(t, "Alice", "POST", endpoint+"/publish", input, 200)
	current, err := f.store.DesktopApplication(context.Background())
	if err != nil || current.Revision != 1 || current.Version != release.Version {
		t.Fatal("actual artifact publication failed", err)
	}
	public := socialDataV2(f.request(t, "", "GET", "/api/v1/launcher/application", nil, 200))
	if public["release"] == nil {
		t.Fatal("published artifact is not readable")
	}
}

//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

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
	release, err := verifyDesktopPublication(raw, desktopApplicationPublicKey)
	if err != nil {
		t.Fatal(err)
	}
	f := newSocialFixtureV2(t)
	if err = f.store.Grant(context.Background(), f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	objects := newApplicationObjects(t)
	runtime := objects.runtime()
	runtime.publicURL = strings.TrimSuffix(release.Package.URL, "/"+desktopApplicationObjectKey(release))
	f.app.desktop = runtime
	bad := objects.directory + "/incomplete.zip"
	if err := os.WriteFile(bad, []byte("incomplete"), 0600); err != nil {
		t.Fatal(err)
	}
	key := desktopApplicationObjectKey(release)
	objects.seed(key, bad, 10, release.Package.SHA256)
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
	objects.seed(key, packagePath, release.Package.Size, release.Package.SHA256)
	if release.UpdaterProtocol == 2 {
		objects.fail.Store(true)
		f.request(t, "Alice", "POST", endpoint+"/publish", input, 409)
		before, _ = f.store.DesktopApplication(context.Background())
		if before.Revision != 0 {
			t.Fatal("failed content preparation changed publication")
		}
		objects.fail.Store(false)
	}
	f.request(t, "Alice", "POST", endpoint+"/publish", input, 200)
	f.request(t, "Alice", "POST", endpoint+"/publish", input, 200)
	current, err := f.store.DesktopApplication(context.Background())
	if err != nil || current.Revision != 1 || current.Version != release.Version {
		t.Fatal("actual artifact publication failed", err)
	}
	for _, query := range []string{"", "?updaterProtocol=2", "?updaterProtocol=999"} {
		public := socialDataV2(f.request(t, "", "GET", "/api/v1/launcher/application"+query, nil, 200))
		selected, _ := json.Marshal(public["release"])
		value, err := verifyDesktopApplication(selected, desktopApplicationPublicKey)
		expected := 1
		if query == "?updaterProtocol=2" {
			expected = release.UpdaterProtocol
		}
		if err != nil || value.UpdaterProtocol != expected {
			t.Fatal("incompatible public release", query, value.UpdaterProtocol, err)
		}
	}
	t.Logf("Verified signed %s artifact (%d bytes): permissions, incomplete upload, preparation failure, idempotency and protocol negotiation", release.Version, release.Package.Size)
}

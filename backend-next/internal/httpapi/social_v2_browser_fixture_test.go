//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/objectstorage"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

// Opt-in browser harness. It uses only testdb.New's loopback random schema and
// test identities. No production DSN, Core node or user data is read or mutated.
func TestSocialBrowserFixtureV2(t *testing.T) {
	if os.Getenv("DEUTERIUM_BROWSER_QA") != "1" {
		t.Skip("opt-in browser fixture")
	}
	output := os.Getenv("DEUTERIUM_BROWSER_QA_OUTPUT")
	origin := os.Getenv("DEUTERIUM_BROWSER_QA_ORIGIN")
	if output == "" || origin != "http://127.0.0.1:5221" {
		t.Fatal("explicit loopback browser fixture output/origin required")
	}
	for key, value := range map[string]string{"DEUTERIUM_S3_ENDPOINT": "https://storage.example.invalid", "DEUTERIUM_S3_REGION": "test", "DEUTERIUM_S3_BUCKET": "browser-fixture", "DEUTERIUM_S3_PREFIX": "deuterium-test/", "DEUTERIUM_S3_ACCESS_KEY": "test-only-key", "DEUTERIUM_S3_SECRET_KEY": "test-only-secret"} {
		t.Setenv(key, value)
	}
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	f.app.Config.PublicOrigin = origin
	f.app.Config.Nodes = []config.Node{{ID: "amiya", Chat: true, ClaimEnabled: true, InventoryDomain: "survival"}}
	f.app.Hub.JoinNode("amiya")
	defer f.app.Hub.LeaveNode("amiya")
	if err := f.store.Grant(ctx, f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.store.PublishAppChat(ctx, f.users["Bob"], "browser-public", "本机浏览器公共引用验证", nil); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(output, 0700); err != nil {
		t.Fatal(err)
	}
	writeJSONFile := func(name string, value any) {
		t.Helper()
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(output, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"Alice", "Bob"} {
		token := identity.Secret()
		expires := time.Now().Add(15 * time.Minute)
		if err := f.store.CreateSession(ctx, f.users[name], store.Digest([]byte(token)), "web", identity.Secret(), expires, ""); err != nil {
			t.Fatal(err)
		}
		writeJSONFile(name+"-state.json", map[string]any{"cookies": []any{map[string]any{"name": "deuterium_dev_session", "value": token, "domain": "127.0.0.1", "path": "/", "expires": expires.Unix(), "httpOnly": true, "secure": false, "sameSite": "Lax"}}, "origins": []any{}})
	}
	media := map[string]store.AssetUploadV2{}
	for _, purpose := range []string{"AVATAR", "MARKET_PHOTO", "ANNOUNCEMENT_MEDIA", "STORE_MEDIA"} {
		assetID := store.ID("asset_")
		u, err := f.store.CreateAssetUploadV2(ctx, store.AssetUploadV2{UploadID: store.ID("upload_"), AssetID: assetID, UserID: f.users["Alice"].ID, ClientRequestID: assetID, Fingerprint: store.Digest([]byte(assetID)), Purpose: purpose, BusinessType: "FIXTURE", ObjectKey: "deuterium-test/" + assetID + ".png", ContentType: "image/png", SizeBytes: 100, MD5: "AAAAAAAAAAAAAAAAAAAAAA=="})
		if err != nil {
			t.Fatal(err)
		}
		if ok, err := f.store.BeginAssetVerificationV2(ctx, u); err != nil || !ok {
			t.Fatal(err)
		}
		if err = f.store.FinishAssetVerificationV2(ctx, u, objectstorage.VerifiedImage{Width: 8, Height: 8, SHA256: store.Digest([]byte(assetID))}, nil); err != nil {
			t.Fatal(err)
		}
		media[purpose] = u
	}
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.RGBA{65, 124, 210, 255})
		}
	}
	file, err := os.Create(filepath.Join(output, "fixture.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
	file.Close()
	finished := make(chan struct{})
	var once sync.Once
	finishToken := identity.Secret()
	normal := f.app.Handler()
	f.http.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/__qa__/finish" && r.Method == "POST" && r.Header.Get("X-QA-Finish") == finishToken {
			w.WriteHeader(204)
			once.Do(func() { close(finished) })
			return
		}
		if r.URL.Path == "/api/v1/assets/uploads" && r.Method == "POST" {
			u, err := f.app.authenticate(r)
			if err != nil {
				failError(w, r, err)
				return
			}
			var input assetUploadInputV2
			if body(w, r, &input) != nil || !validateAssetInputV2(input) {
				failure(w, r, 400, "INVALID_REQUEST", "Invalid fixture upload")
				return
			}
			asset, ok := media[input.Purpose]
			if !ok || asset.UserID != u.User.ID {
				failure(w, r, 403, "FORBIDDEN", "Not a fixture media owner")
				return
			}
			view, err := f.store.AssetV2(r.Context(), u.User.ID, asset.AssetID)
			if err != nil {
				assetErrorV2(w, r, err)
				return
			}
			v2Success(w, r, map[string]any{"uploadId": asset.UploadID, "assetId": asset.AssetID, "status": "READY", "asset": view})
			return
		}
		normal.ServeHTTP(w, r)
	})
	writeJSONFile("connection.json", map[string]any{"backendOrigin": f.http.URL, "webOrigin": origin, "finishToken": finishToken})
	t.Log("Loopback browser fixture ready; connection details are in the ignored output directory.")
	select {
	case <-finished:
	case <-time.After(12 * time.Minute):
		t.Fatal("browser fixture timed out")
	}
}

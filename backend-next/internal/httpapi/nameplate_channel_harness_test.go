//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Opt-in isolated Youer/Core/LuckPerms channel harness; only disposable local testdb data.
func TestNameplateChannelHarness(t *testing.T) {
	out := os.Getenv("DEUTERIUM_NAMEPLATE_QA_OUTPUT")
	if out == "" {
		t.Skip("opt-in native channel harness")
	}
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	if _, err := f.store.DB.Exec("INSERT INTO identity_permissions VALUES(?,'platform.admin')", f.users["Alice"].ID); err != nil {
		t.Fatal(err)
	}
	titles := map[string]store.GameTitle{}
	for _, name := range []string{"public_title", "private_title"} {
		raw, err := f.store.ChangeGameTitle(ctx, f.users["Alice"].ID, "", store.GameTitleChange{ClientRequestID: "create-" + name, InternalID: name, DisplayName: "#d9afd9测试#97d9e1铭牌", Description: "介绍与获取方式：本地通道验收。", Public: name == "public_title", Weight: 10, Status: "PUBLISHED"})
		if err != nil {
			t.Fatal(err)
		}
		var title store.GameTitle
		if err = json.Unmarshal(raw, &title); err != nil {
			t.Fatal(err)
		}
		titles[name] = title
	}
	token := "local-nameplate-node-012345678901234567890"
	f.app.Config.Nodes = []config.Node{{ID: "amiya", Chat: true, TokenSHA256: store.Digest([]byte(token)), ItemPrefix: "deuterium", InventoryDomain: "survival", CompatibilityProfile: "ix-main-1_21_1"}}
	done := make(chan struct{})
	var once sync.Once
	normal := f.app.Handler()
	f.http.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/__qa__/finish" && r.Method == "POST" {
			once.Do(func() { close(done) })
			w.WriteHeader(204)
			return
		}
		if r.URL.Path == "/__qa__/status" {
			profiles, _ := f.store.GameTitleProfiles(r.Context())
			writeJSON(w, 200, map[string]any{"online": f.app.Core.Online("amiya"), "profiles": profiles})
			return
		}
		normal.ServeHTTP(w, r)
	})
	os.MkdirAll(out, 0700)
	raw, _ := json.MarshalIndent(map[string]any{"apiBase": f.http.URL, "bridge": "ws" + strings.TrimPrefix(f.http.URL, "http") + "/bridge/v1/connect", "nodeToken": token, "titles": titles}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "harness.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	t.Log("nameplate channel harness ready")
	select {
	case <-done:
	case <-time.After(12 * time.Minute):
		t.Fatal("channel harness timeout")
	}
}

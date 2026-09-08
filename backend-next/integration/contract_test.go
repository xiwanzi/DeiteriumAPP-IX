//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

// Optional response capture for scripts/validate_contract.py against the App's
// full source OpenAPI. Fixtures contain synthetic accounts only.
func TestCaptureImplementedContractResponses(t *testing.T) {
	f := newFixture(t)
	response, auth := f.request(t, "POST", "/api/v1/account/login", map[string]string{"account": "Alice", "password": password}, nil, nil)
	if response.StatusCode != 200 {
		t.Fatal(auth)
	}
	token := auth["data"].(map[string]any)["token"].(string)
	_, profile := f.request(t, "GET", "/api/v1/account/me", nil, bearer(token), nil)
	chat := store.CoreChat{PlayerUUID: aliceUUID, GameID: "Alice", Content: "contract fixture"}
	payload, _ := json.Marshal(chat)
	if _, _, err := f.store.CoreEvent(context.Background(), "amiya", "contract-message", "chat.public.event", payload, &chat, nil); err != nil {
		t.Fatal(err)
	}
	_, messages := f.request(t, "GET", "/api/v1/chat/messages", nil, bearer(token), nil)
	_, invalid := f.request(t, "GET", "/api/v1/account/me", nil, nil, nil)
	_, logout := f.request(t, "POST", "/api/v1/account/logout", map[string]any{}, bearer(token), nil)
	// Session is revoked; output never retains even a synthetic usable token.
	auth["data"].(map[string]any)["token"] = "revoked-synthetic-contract-token"
	samples := map[string]any{"AuthResponse": auth, "UserProfileResponse": profile, "ChatMessagesResponse": messages, "ErrorResponse": invalid, "LogoutResponse": logout}
	if dir := os.Getenv("DEUTERIUM_CONTRACT_SAMPLES"); dir != "" {
		if !filepath.IsAbs(dir) {
			t.Fatal("contract output directory must be absolute")
		}
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		data, _ := json.MarshalIndent(samples, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, "responses.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

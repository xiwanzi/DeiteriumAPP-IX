package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestYSMIdentityRequiresDedicatedServiceCredential(t *testing.T) {
	token := strings.Repeat("s", 48)
	sum := sha256.Sum256([]byte(token))
	for _, test := range []struct {
		digest, header, uuid string
		want                 int
	}{
		{"", "Bearer " + token, "", 401},
		{hex.EncodeToString(sum[:]), "", "", 401},
		{hex.EncodeToString(sum[:]), "Bearer website-session", "", 401},
		{hex.EncodeToString(sum[:]), "Basic " + token, "", 401},
		{hex.EncodeToString(sum[:]), "Bearer " + token, "PlayerName", 400},
		{hex.EncodeToString(sum[:]), "Bearer " + token, "00000000000000000000000000000001", 400},
	} {
		server := &Server{Config: config.Config{YSMIdentityTokenSHA256: test.digest}}
		request := httptest.NewRequest("GET", "/internal/v1/ysm/identities/x", nil)
		request.Header.Set("Authorization", test.header)
		request.SetPathValue("uuid", test.uuid)
		response := httptest.NewRecorder()
		server.ysmIdentity(response, request)
		if response.Code != test.want {
			t.Fatalf("got %d want %d", response.Code, test.want)
		}
	}
}

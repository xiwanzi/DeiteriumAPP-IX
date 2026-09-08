package httpapi

import (
	"net/http/httptest"
	"testing"
)

func TestSocialCursorCannotCrossUsersOrConversationScopes(t *testing.T) {
	encoded := encodeSocialCursorV2(socialCursorV2{Scope: "messages:alice:ab", Before: 17})
	request := httptest.NewRequest("GET", "/api/v1/chat/conversations/ab/messages?cursor="+encoded+"&limit=2", nil)
	limit, cursor, err := socialPageV2(request, "messages:alice:ab")
	if err != nil || limit != 2 || cursor.Before != 17 {
		t.Fatal("valid cursor rejected")
	}
	if _, _, err = socialPageV2(request, "messages:carol:ab"); err == nil {
		t.Fatal("another user's cursor accepted")
	}
	if _, _, err = socialPageV2(request, "messages:alice:ac"); err == nil {
		t.Fatal("another conversation's cursor accepted")
	}
	for _, query := range []string{"?limit=0", "?limit=101", "?limit=word", "?cursor=invalid"} {
		if _, _, err = socialPageV2(httptest.NewRequest("GET", "/"+query, nil), "scope"); err == nil {
			t.Fatalf("invalid pagination accepted: %s", query)
		}
	}
}

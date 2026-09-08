package store

import (
	"strings"
	"testing"
)

func TestSocialInputBoundsCountUnicodeAndRejectControlCharacters(t *testing.T) {
	if !ValidSocialText(strings.Repeat("中文", 128), 1, 256) {
		t.Fatal("valid Unicode text rejected")
	}
	for _, text := range []string{strings.Repeat("字", 257), " ", "hello\x00world", "hello\rworld"} {
		if ValidSocialText(text, 1, 256) {
			t.Fatalf("invalid message accepted: %q", text)
		}
	}
	for _, id := range []string{"", strings.Repeat("x", 129), "../private", "x y", "中文", "x\n"} {
		if ValidSocialID(id) {
			t.Fatal("invalid opaque identifier accepted")
		}
	}
	if !ValidSocialID("message-123_abc") {
		t.Fatal("valid identifier rejected")
	}
}
func TestAnnouncementRequiresBoundedTypedContent(t *testing.T) {
	value := SocialAnnouncementWrite{ClientRequestID: "request-1", Title: "维护说明", Summary: "周末维护", Priority: "NORMAL", ContentBlocks: []SocialContentBlock{{BlockID: "p1", Type: "PARAGRAPH", Text: "请保存进度。"}}}
	if validateSocialAnnouncement(value) != nil {
		t.Fatal("valid announcement rejected")
	}
	value.ContentBlocks = append(value.ContentBlocks, value.ContentBlocks[0])
	if validateSocialAnnouncement(value) == nil {
		t.Fatal("duplicate block IDs accepted")
	}
	value.ContentBlocks = []SocialContentBlock{{BlockID: "p1", Type: "HTML", Text: "<script>alert(1)</script>"}}
	if validateSocialAnnouncement(value) == nil {
		t.Fatal("unsupported HTML content type accepted")
	}
	value.ContentBlocks = []SocialContentBlock{{BlockID: "p1", Type: "IMAGE"}}
	if validateSocialAnnouncement(value) == nil {
		t.Fatal("unbound image accepted")
	}
}

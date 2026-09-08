package store

import "testing"

func TestPublicChatMetadataFingerprintPreservesLegacyAndSetSemantics(t *testing.T) {
	legacy, _, err := publicChatFingerprintV2("hello", "", nil)
	if err != nil || legacy != Digest([]byte("hello")) {
		t.Fatal("legacy idempotency fingerprint changed")
	}
	first, _, err := publicChatFingerprintV2("hello", "message-1", []string{"player_Bob", "player_Carol"})
	if err != nil {
		t.Fatal(err)
	}
	ordered, _, _ := publicChatFingerprintV2("hello", "message-1", []string{"player_Carol", "player_Bob", "player_Bob"})
	if ordered != first {
		t.Fatal("mention ordering changed a semantic set")
	}
	changedReply, _, _ := publicChatFingerprintV2("hello", "message-2", []string{"player_Bob", "player_Carol"})
	if changedReply == first {
		t.Fatal("reply omitted from idempotency fingerprint")
	}
	changedMentions, _, _ := publicChatFingerprintV2("hello", "message-1", []string{"player_Bob"})
	if changedMentions == first {
		t.Fatal("mentions omitted from idempotency fingerprint")
	}
	if _, _, err := publicChatFingerprintV2("hello", "", []string{"../foreign"}); err == nil {
		t.Fatal("invalid player reference accepted")
	}
}

package identity

import (
	"strings"
	"testing"
)

func TestPasswordHashAndBounds(t *testing.T) {
	encoded := Hash("correct horse 123")
	if !Verify(encoded, "correct horse 123") || Verify(encoded, "wrong horse 123") {
		t.Fatal("password verification mismatch")
	}
	if NeedsUpgrade(encoded) {
		t.Fatal("new hash should not need upgrading")
	}
	for _, invalid := range []string{
		strings.Replace(encoded, "m=65536", "m=2147483647", 1),
		strings.Replace(encoded, "t=3", "t=0", 1),
		strings.Replace(encoded, "p=2", "p=256", 1),
		strings.Replace(encoded, "v=19", "v=16", 1),
		encoded + "$extra", "$bcrypt$irrelevant",
	} {
		if ValidateHash(invalid) == nil {
			t.Fatal("unsupported hash accepted")
		}
	}
}

func TestLegacyImportInputRejectsTrailingJSON(t *testing.T) {
	if _, err := ReadLegacy(strings.NewReader(`{} {}`)); err == nil {
		t.Fatal("trailing JSON accepted")
	}
	if _, err := ReadLegacy(strings.NewReader(`{"password":"plaintext"}`)); err == nil {
		t.Fatal("unknown credential field accepted")
	}
}

func BenchmarkArgon2Verify(b *testing.B) {
	encoded := Hash("correct horse 123")
	b.ResetTimer()
	for b.Loop() {
		if !Verify(encoded, "correct horse 123") {
			b.Fatal("verification failed")
		}
	}
}

package identity

import "testing"

func TestLegacyJavaArgon2Fixture(t *testing.T) {
	// Produced by testdata/LegacyArgon2Fixture.java using argon2-jvm 2.11.
	const encoded = "$argon2i$v=19$m=65536,t=3,p=2$o+jTlhl/fSdGLpHQ81nOng$kXV9sfxMH/x7dtnoJCG0EPRudDCmCUlzVFElYGthqzY"
	if !Verify(encoded, "migration-password-123") || Verify(encoded, "incorrect-password-123") {
		t.Fatal("legacy JVM password compatibility failed")
	}
	if !NeedsUpgrade(encoded) {
		t.Fatal("legacy Argon2i should upgrade after successful login")
	}
}

package httpapi

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func signedDesktopApplication(t *testing.T, mutate func(*desktopApplicationRelease)) (json.RawMessage, ed25519.PublicKey) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	value := desktopApplicationRelease{SchemaVersion: 1, Product: "DLauncher", Platform: "windows-x64", Version: "0.6.0", VersionCode: 6000, UpdaterProtocol: 1, Notes: "隔离的签名测试", PublishedAt: "2026-09-13T00:00:00Z"}
	value.Package.Size = 100
	value.Package.SHA256 = strings.Repeat("a", 64)
	value.Package.URL = "https://objects.example/" + desktopApplicationObjectKey(value)
	if mutate != nil {
		mutate(&value)
	}
	payload, _ := json.Marshal(value)
	raw, _ := json.Marshal(desktopApplicationEnvelope{Payload: base64.StdEncoding.EncodeToString(payload), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, payload))})
	return raw, public
}

func TestDesktopApplicationSignatureAndTamper(t *testing.T) {
	raw, key := signedDesktopApplication(t, nil)
	value, err := verifyDesktopApplication(raw, key)
	if err != nil || value.Version != "0.6.0" || value.VersionCode != 6000 {
		t.Fatal("valid fixture signature rejected", err)
	}
	if _, err = verifyDesktopApplication(raw, desktopApplicationPublicKey); err == nil {
		t.Fatal("fixture signer was accepted as the production signer")
	}
	var envelope desktopApplicationEnvelope
	json.Unmarshal(raw, &envelope)
	payload, _ := base64.StdEncoding.DecodeString(envelope.Payload)
	envelope.Payload = base64.StdEncoding.EncodeToString([]byte(strings.ReplaceAll(string(payload), "0.6.0", "0.7.0")))
	raw, _ = json.Marshal(envelope)
	if _, err = verifyDesktopApplication(raw, key); err == nil {
		t.Fatal("tampered signed payload accepted")
	}
}

func TestDesktopApplicationRejectsInvalidSignedReleases(t *testing.T) {
	for _, mutate := range []func(*desktopApplicationRelease){
		func(v *desktopApplicationRelease) { v.Platform = "linux" },
		func(v *desktopApplicationRelease) { v.VersionCode = 6001 },
		func(v *desktopApplicationRelease) { v.Version = "0.06.0" },
		func(v *desktopApplicationRelease) { v.UpdaterProtocol = 2 },
		func(v *desktopApplicationRelease) { v.Package.Size = desktopApplicationLimit + 1 },
		func(v *desktopApplicationRelease) { v.Package.URL = "file:///DLauncher.exe" },
		func(v *desktopApplicationRelease) {
			v.Package.URL = strings.Replace(v.Package.URL, "https://", "https://user:secret@", 1)
		},
		func(v *desktopApplicationRelease) { v.Package.URL += "?token=not-allowed" },
		func(v *desktopApplicationRelease) { v.Package.URL = "https://objects.example/wrong.zip" },
		func(v *desktopApplicationRelease) { v.PublishedAt = "2026-99-13T00:00:00Z" },
	} {
		raw, key := signedDesktopApplication(t, mutate)
		if _, err := verifyDesktopApplication(raw, key); err == nil {
			t.Fatal("invalid signed release was accepted", string(raw))
		}
	}
}

func TestDesktopApplicationEnvelopeUsesExactUniqueFields(t *testing.T) {
	raw, key := signedDesktopApplication(t, nil)
	for _, invalid := range []string{
		strings.Replace(string(raw), `"payload"`, `"Payload"`, 1),
		strings.Replace(string(raw), `"signature":`, `"payload":"duplicate","signature":`, 1),
	} {
		if _, err := verifyDesktopApplication([]byte(invalid), key); err == nil {
			t.Fatal("ambiguous envelope accepted")
		}
	}
}

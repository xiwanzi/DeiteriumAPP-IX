package httpapi

import (
	"os"
	"testing"
)

func TestOperatorRotatedSignedPackage(t *testing.T) {
	fixture := os.Getenv("DLAUNCHER_SIGNED_FIXTURE")
	if fixture == "" {
		t.Skip("set DLAUNCHER_SIGNED_FIXTURE for release signing verification")
	}
	b, e := os.ReadFile(fixture)
	if e != nil {
		t.Fatal(e)
	}
	r, e := verifyDesktopPublication(b, desktopApplicationPublicKey)
	if e != nil || r.Version != "0.6.11" || r.UpdaterProtocol != 2 {
		t.Fatalf("publication rejected: %v", e)
	}
}

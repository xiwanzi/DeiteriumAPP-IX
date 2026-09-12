package config

import "testing"

func TestAdmissionURLKeepsTheConfiguredOrigin(t *testing.T) {
	c := Config{Listen: "127.0.0.1:8080", PublicOrigin: "https://47.103.99.34", AdmissionPublicOrigin: "https://47.103.99.34", AdmissionApplicationURL: "https://47.103.99.34/admission/"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if c.AdmissionURL() != c.AdmissionApplicationURL {
		t.Fatal("application path lost")
	}
	for _, invalid := range []string{"https://unrelated.invalid/admission/", "javascript:alert(1)", "https://user@47.103.99.34/admission/", "https://47.103.99.34/admission/?token=secret"} {
		c.AdmissionApplicationURL = invalid
		if c.Validate() == nil {
			t.Fatalf("accepted %q", invalid)
		}
	}
	c.AdmissionApplicationURL = ""
	if c.AdmissionURL() != c.AdmissionPublicOrigin {
		t.Fatal("missing origin fallback")
	}
}

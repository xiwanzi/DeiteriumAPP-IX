package config

import "testing"

func TestYSMOnlyAcceptsLiteralLoopbackOrigins(t *testing.T) {
	for _, raw := range []string{"http://127.0.0.1:17656", "http://[::1]:17656"} {
		if err := (Config{YSMAdminURL: raw, YSMAdminOrigin: "http://127.0.0.1:17657"}).ValidateYSM(); err != nil {
			t.Fatal(err)
		}
	}
	for _, raw := range []string{"http://localhost:17656", "http://192.168.1.1:17656", "https://example.com:443", "http://127.0.0.1", "http://127.0.0.1:0", "http://127.0.0.1:99999", "http://u:p@127.0.0.1:17656", "http://127.0.0.1:17656/api", "http://127.0.0.1:17656?", "http://127.0.0.1:17656#"} {
		if (Config{YSMAdminURL: raw}).ValidateYSM() == nil {
			t.Fatalf("accepted unsafe origin %q", raw)
		}
		if (Config{YSMAdminURL: "http://127.0.0.1:17656", YSMAdminOrigin: raw}).ValidateYSM() == nil {
			t.Fatalf("accepted unsafe Host %q", raw)
		}
	}
	if (Config{YSMAdminOrigin: "http://127.0.0.1:17656"}).ValidateYSM() == nil {
		t.Fatal("origin without connection accepted")
	}
}

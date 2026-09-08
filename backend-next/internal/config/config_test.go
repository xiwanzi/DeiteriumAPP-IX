package config

import (
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"testing"
)

func TestConfigurationRejectsUnsafeOriginsAndSharedCredentials(t *testing.T) {
	c := Config{Listen: "127.0.0.1:8080", PublicOrigin: "https://community.example.invalid"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, origin := range []string{"http://community.example.invalid", "https://community.example.invalid/path", "https://user:password@community.example.invalid", "https://community.example.invalid?query"} {
		c.PublicOrigin = origin
		if c.Validate() == nil {
			t.Fatal("unsafe origin accepted")
		}
	}
	c.PublicOrigin = "http://127.0.0.1:8080"
	c.Development = true
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Listen = "0.0.0.0:8080"
	if c.Validate() == nil {
		t.Fatal("public insecure development listener accepted")
	}
	c.Listen = "127.0.0.1:8080"
	hash := store.Digest([]byte("a-long-test-node-secret-1234567890"))
	c.Nodes = []Node{{ID: "amiya", TokenSHA256: hash, InventoryDomain: "survival"}, {ID: "odyssey", TokenSHA256: hash, InventoryDomain: "survival"}}
	if c.Validate() == nil {
		t.Fatal("shared node secret accepted")
	}
}

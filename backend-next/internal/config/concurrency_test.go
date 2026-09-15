package config

import "testing"

func TestConcurrencyDefaultsAndInvalidLimits(t *testing.T) {
	c := (Concurrency{}).WithDefaults()
	if c.HTTPRequests != 256 || c.RealtimeConnections != 1024 || c.AIStreams != 64 || c.ControlRequests != 64 {
		t.Fatal(c)
	}
	for _, invalid := range []Concurrency{{HTTPRequests: -1}, {RealtimeConnections: 16385}, {AIStreams: -1}, {ControlRequests: -1}} {
		if invalid.Validate() == nil {
			t.Fatalf("invalid limit accepted: %+v", invalid)
		}
	}
	if err := (Concurrency{HTTPRequests: 100, RealtimeConnections: 500}).Validate(); err != nil {
		t.Fatal(err)
	}
}

package store

import (
	"testing"
	"time"
)

func TestAIQuotaWindowUsesEpochV2(t *testing.T) {
	now := time.Date(2026, 9, 8, 7, 59, 59, 0, time.FixedZone("UTC+8", 8*3600))
	if want := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC); !AIWindowStartV2(now, 24).Equal(want) {
		t.Fatalf("wrong day window: %v", AIWindowStartV2(now, 24))
	}
	for _, hours := range []int{5, 24} {
		start := AIWindowStartV2(now, hours)
		if start.Unix()%int64(hours*3600) != 0 || start.After(now) || !start.Add(time.Duration(hours)*time.Hour).After(now) {
			t.Fatal("invalid epoch window")
		}
	}
}

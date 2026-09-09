package store

import (
	"testing"
	"time"
)

func TestAIUpgradeRoundedDaysAndExactCentsV207(t *testing.T) {
	for _, tt := range []struct {
		name, old, next string
		remaining       time.Duration
		days            int64
		amount          string
	}{
		{"partial day", "12.50", "42.50", 7*24*time.Hour + 12*time.Hour, 8, "16.00"},
		{"exact day", "12.50", "42.50", 24 * time.Hour, 1, "2.00"},
		{"one tick over", "12.50", "42.50", 24*time.Hour + time.Nanosecond, 2, "4.00"},
		{"last tick", "12.50", "42.50", time.Nanosecond, 1, "2.00"},
		{"round up", "1.00", "2.00", time.Hour, 1, "0.07"},
		{"round down", "1.00", "1.95", time.Hour, 1, "0.06"},
		{"renewed balance", "12.50", "42.50", 30 * 24 * time.Hour, 30, "60.00"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			amount, days, err := aiUpgradeAmountV207(tt.old, tt.next, tt.remaining)
			if err != nil || amount != tt.amount || days != tt.days {
				t.Fatalf("%s %d %v", amount, days, err)
			}
		})
	}
	for _, tt := range []struct {
		old, next string
		remaining time.Duration
	}{{"10.00", "9.00", time.Hour}, {"10.00", "10.00", time.Hour}, {"10.00", "20.00", 0}, {"1.00", "1.01", time.Hour}} {
		if _, _, err := aiUpgradeAmountV207(tt.old, tt.next, tt.remaining); err == nil {
			t.Fatal("invalid charge accepted")
		}
	}
}

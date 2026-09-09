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
		{"partial day", "12.50", "42.50", 7*24*time.Hour + 12*time.Hour, 8, "8.00"},
		{"exact day", "12.50", "42.50", 24 * time.Hour, 1, "5.00"},
		{"one tick over", "12.50", "42.50", 24*time.Hour + time.Nanosecond, 2, "5.00"},
		{"last tick", "12.50", "42.50", time.Nanosecond, 1, "5.00"},
		{"round up", "1.00", "2.00", time.Hour, 1, "0.17"},
		{"round down", "1.00", "1.92", time.Hour, 1, "0.15"},
		{"renewed balance", "12.50", "42.50", 60 * 24 * time.Hour, 60, "60.00"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			amount, days, err := aiUpgradeAmountV207(tt.old, tt.next, tt.remaining, 30)
			if err != nil || amount != tt.amount || days != tt.days {
				t.Fatalf("%s %d %v", amount, days, err)
			}
		})
	}
	for _, tt := range []struct {
		old, next string
		remaining time.Duration
	}{{"10.00", "9.00", time.Hour}, {"10.00", "10.00", time.Hour}, {"10.00", "20.00", 0}, {"1.00", "1.01", time.Hour}} {
		if _, _, err := aiUpgradeAmountV207(tt.old, tt.next, tt.remaining, 30); err == nil {
			t.Fatal("invalid charge accepted")
		}
	}
}

func TestAIUpgradeFirstDayDeclineFiveDayFloorAndShortCycleV207(t *testing.T) {
	for _, tt := range []struct {
		days int
		want string
	}{{30, "15000.00"}, {29, "14500.00"}, {15, "7500.00"}, {6, "3000.00"}, {5, "2500.00"}, {4, "2500.00"}, {1, "2500.00"}} {
		got, _, err := aiUpgradeAmountV207("5000.00", "20000.00", time.Duration(tt.days)*24*time.Hour, 30)
		if err != nil || got != tt.want {
			t.Fatalf("remaining=%d got=%s err=%v", tt.days, got, err)
		}
	}
	got, _, err := aiUpgradeAmountV207("5.00", "20.00", time.Hour, 3)
	if err != nil || got != "15.00" {
		t.Fatal("short cycle exceeded the full difference", got, err)
	}
	if _, _, err = aiUpgradeAmountV207("5.00", "20.00", time.Hour, 0); err == nil {
		t.Fatal("invalid purchased period accepted")
	}
}

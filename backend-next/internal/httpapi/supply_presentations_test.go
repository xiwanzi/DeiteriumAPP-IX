package httpapi

import (
	"context"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/supply"
)

func TestPublicSupplyCatalogDeltaSkipsUnchangedPoolsAndIncludesNewFields(t *testing.T) {
	srv := &Server{Config: config.Config{PublicOrigin: "https://example.invalid"}}
	samples := store.SampleSupplyPools(time.Now().UTC())
	samples[0].Definition.Labels.FeaturedTitle = "限定主推测试"
	samples[0].Definition.Labels.FeaturedLead = "本池概率 {goldProb}"
	pools := make([]store.SupplyPool, 0, len(samples))
	for _, sample := range samples {
		pools = append(pools, store.SupplyPool{
			ID:                sample.ID,
			Version:           1,
			PublishedRevision: 2,
			Enabled:           true,
			Definition:        sample.Definition,
		})
	}
	full, art, deltaMode, err := srv.publicSupplyCatalogDelta(context.Background(), pools, nil, 0)
	if err != nil || deltaMode {
		t.Fatalf("full catalog failed: deltaMode=%v err=%v", deltaMode, err)
	}
	if len(full["order"].([]string)) != 5 || len(full["pools"].(map[string]any)) != 5 || len(art) == 0 {
		t.Fatalf("unexpected full catalog sizes: order=%d pools=%d art=%d", len(full["order"].([]string)), len(full["pools"].(map[string]any)), len(art))
	}
	up80 := full["pools"].(map[string]any)["sp_sample_up_191_80"].(map[string]any)
	if up80["label"] != "限定 · 80" {
		t.Fatalf("unexpected rail label (should use pool kind badge): %v", up80["label"])
	}
	std80 := full["pools"].(map[string]any)["sp_sample_standard_80"].(map[string]any)
	if std80["label"] != "常驻" {
		t.Fatalf("unexpected standard rail label: %v", std80["label"])
	}
	if up80["featuredTitle"] != "限定主推测试" || up80["featuredLead"] != "本池概率 {goldProb}" {
		t.Fatalf("missing featured copy fields: %+v", up80)
	}
	if up80["subtitle"] == "" || up80["itemsBound"].(bool) || len(up80["sections"].([]supply.PoolSection)) == 0 {
		t.Fatalf("missing new presentation fields: %+v", up80)
	}

	known := map[string]int64{
		"sp_sample_up_191":    2,
		"sp_sample_up_vector": 2,
		"sp_sample_standard":  2,
	}
	unchanged, unchangedArt, deltaMode, err := srv.publicSupplyCatalogDelta(context.Background(), pools, known, supplyPresentationVersion)
	if err != nil || !deltaMode || !unchanged["unchanged"].(bool) {
		t.Fatalf("expected unchanged delta: deltaMode=%v unchanged=%+v err=%v", deltaMode, unchanged, err)
	}
	if len(unchanged["pools"].(map[string]any)) != 0 || len(unchanged["weapons"].(map[string]any)) != 0 || len(unchangedArt) != 0 {
		t.Fatalf("expected empty delta payload: pools=%d weapons=%d art=%d", len(unchanged["pools"].(map[string]any)), len(unchanged["weapons"].(map[string]any)), len(unchangedArt))
	}

	// Simulate editing 2 characters in sp_sample_standard -> published revision becomes 3
	pools[2].PublishedRevision = 3
	partial, partialArt, deltaMode, err := srv.publicSupplyCatalogDelta(context.Background(), pools, known, supplyPresentationVersion)
	if err != nil || !deltaMode || partial["unchanged"].(bool) {
		t.Fatalf("expected partial delta: %+v err=%v", partial, err)
	}
	changedPools := partial["pools"].(map[string]any)
	if len(changedPools) != 1 || changedPools["sp_sample_standard_80"] == nil {
		t.Fatalf("expected only sp_sample_standard_80 in partial delta: %+v", changedPools)
	}
	if len(partial["order"].([]string)) != 5 || len(partialArt) == 0 {
		t.Fatalf("expected full order and only changed pool art: order=%d art=%d", len(partial["order"].([]string)), len(partialArt))
	}
}

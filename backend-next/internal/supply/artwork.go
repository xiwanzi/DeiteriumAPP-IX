package supply

import "strings"

var ModelKeys = []string{"printstream", "scarlet", "gilded-rain", "perfect-freeze", "tacz-ai_awp", "tacz-m249", "tacz-scar_h", "tacz-m1014", "tacz-ak47", "tacz-m4a1", "tacz-hk_mp5a5", "tacz-glock_17", "tacz-ump45", "tacz-sks_tactical"}

// Bundled artwork names are capabilities, not arbitrary resource paths supplied by a browser.
func BuiltinArtwork() map[string]string {
	out := map[string]string{
		"builtin-hero-up": "pool-up.png", "builtin-hero-vector": "pool-up-vector.png", "builtin-hero-standard": "pool-standard-1.png",
		"builtin-rail-up": "rail-up.png", "builtin-rail-vector": "rail-up-vector.png", "builtin-rail-standard": "rail-standard-1.png",
	}
	for _, key := range ModelKeys {
		out["builtin-detail-"+key] = "weapons/" + key + ".png"
		folder := "fallback/"
		if strings.HasPrefix(key, "tacz-") {
			folder = "weapons/"
		}
		out["builtin-fallback-"+key] = folder + key + ".png"
	}
	return out
}

func (d Definition) AllArtwork() []string {
	seen := map[string]bool{}
	out := []string{}
	appendArt := func(a Artwork) {
		for _, id := range []string{a.Hero, a.Rail, a.Detail, a.Fallback} {
			if id != "" && !seen[id] {
				out = append(out, id)
				seen[id] = true
			}
		}
	}
	appendArt(d.Art)
	for _, reward := range d.Rewards {
		appendArt(reward.Art)
	}
	return out
}

package bridge

import (
	"encoding/json"
	"sort"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

// Presence reads only snapshots from currently connected, caller-selected nodes.
// An empty received snapshot is available; a missing snapshot is not.
func (r *Runtime) Presence(nodes []string) ([]store.CorePlayerIdentity, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	players := make(map[string]store.CorePlayerIdentity)
	available := false
	for _, node := range nodes {
		if r.links[node] == nil {
			continue
		}
		var snapshot struct {
			Players []store.CorePlayerIdentity `json:"players"`
		}
		if json.Unmarshal(r.presence[node], &snapshot) != nil {
			continue
		}
		available = true
		for _, p := range snapshot.Players {
			if _, exists := players[p.PlayerUUID]; !exists {
				p.ServerID, p.Online = node, true
				players[p.PlayerUUID] = p
			}
		}
	}
	result := make([]store.CorePlayerIdentity, 0, len(players))
	for _, p := range players {
		result = append(result, p)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].GameID == result[j].GameID {
			return result[i].PlayerUUID < result[j].PlayerUUID
		}
		return result[i].GameID < result[j].GameID
	})
	return result, available
}

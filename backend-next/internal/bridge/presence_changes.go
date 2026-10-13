package bridge

import (
	"encoding/json"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"time"
)

// Compare authenticated session snapshots before persistence. A session change
// is significant even if its UUID and spelling stay the same.
func (r *Runtime) PresenceChanges(node string, players []store.CorePlayerIdentity) (changed []store.CorePlayerIdentity, affected []string) {
	r.mu.RLock()
	var previous struct {
		Players []store.CorePlayerIdentity `json:"players"`
	}
	_ = json.Unmarshal(r.presence[node], &previous)
	r.mu.RUnlock()
	old := map[string]store.CorePlayerIdentity{}
	for _, p := range previous.Players {
		old[p.PlayerUUID] = p
	}
	for _, p := range players {
		prior, present := old[p.PlayerUUID]
		if !present || prior.GameID != p.GameID || prior.SessionEpoch != p.SessionEpoch {
			p.ServerID = node
			p.Online = true
			changed = append(changed, p)
			affected = append(affected, p.PlayerUUID)
		}
		delete(old, p.PlayerUUID)
	}
	for uuid, p := range old {
		p.ServerID = node
		p.Online = false
		p.LastSeen = time.Now().UnixMilli()
		changed = append(changed, p)
		affected = append(affected, uuid)
	}
	return
}

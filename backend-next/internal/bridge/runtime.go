package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"strings"
	"sync"
)

var ErrNodeOffline = errors.New("core node offline")

type Runtime struct {
	mu       sync.RWMutex
	links    map[string]func(context.Context, string, any) error
	status   map[string]json.RawMessage
	presence map[string]json.RawMessage
	waiters  map[string]map[chan struct{}]struct{}
}

// A duplicate cross-server presence is ambiguous until the old session leaves.
func (r *Runtime) Player(name, uuid string) (p store.CorePlayerIdentity, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for node, raw := range r.presence {
		var snapshot struct {
			Players []struct {
				PlayerUUID string `json:"playerUuid"`
				GameID     string `json:"gameId"`
			} `json:"players"`
		}
		if json.Unmarshal(raw, &snapshot) != nil {
			continue
		}
		for _, v := range snapshot.Players {
			if (uuid != "" && v.PlayerUUID == uuid) || (name != "" && strings.EqualFold(v.GameID, name)) {
				if p.PlayerUUID != "" {
					return store.CorePlayerIdentity{}, errors.New("ambiguous player presence")
				}
				p = store.CorePlayerIdentity{PlayerUUID: v.PlayerUUID, GameID: v.GameID, ServerID: node, Online: true}
			}
		}
	}
	return
}

func NewRuntime() *Runtime {
	return &Runtime{links: map[string]func(context.Context, string, any) error{}, status: map[string]json.RawMessage{}, presence: map[string]json.RawMessage{}, waiters: map[string]map[chan struct{}]struct{}{}}
}
func (r *Runtime) Connect(node string, send func(context.Context, string, any) error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.links[node] = send
}
func (r *Runtime) Disconnect(node string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.links, node)
	delete(r.status, node)
	delete(r.presence, node)
}
func (r *Runtime) Send(ctx context.Context, node string, payload any) error {
	r.mu.RLock()
	send := r.links[node]
	r.mu.RUnlock()
	if send == nil {
		return ErrNodeOffline
	}
	return send(ctx, "core.command", payload)
}
func (r *Runtime) Online(node string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.links[node] != nil
}
func (r *Runtime) Status(node string) json.RawMessage {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append(json.RawMessage(nil), r.status[node]...)
}
func (r *Runtime) SetStatus(node string, payload []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.links[node] != nil {
		r.status[node] = append([]byte(nil), payload...)
	}
}
func (r *Runtime) SetPresence(node string, payload []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.links[node] != nil {
		r.presence[node] = append([]byte(nil), payload...)
	}
}
func (r *Runtime) Watch(id string) (chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	r.mu.Lock()
	if r.waiters[id] == nil {
		r.waiters[id] = map[chan struct{}]struct{}{}
	}
	r.waiters[id][ch] = struct{}{}
	r.mu.Unlock()
	return ch, func() {
		r.mu.Lock()
		delete(r.waiters[id], ch)
		if len(r.waiters[id]) == 0 {
			delete(r.waiters, id)
		}
		r.mu.Unlock()
	}
}
func (r *Runtime) Notify(id string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for ch := range r.waiters[id] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

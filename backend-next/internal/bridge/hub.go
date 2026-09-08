package bridge

import "sync"

// Wakeups are hints only. SQL is the source of truth; periodic scans cover a
// commit followed by a crash before notification. Memory never holds messages.
type Hub struct {
	mu        sync.Mutex
	listeners map[chan struct{}]struct{}
	nodes     map[string]bool
}

func NewHub() *Hub { return &Hub{listeners: map[chan struct{}]struct{}{}, nodes: map[string]bool{}} }
func (h *Hub) Subscribe() (chan struct{}, func()) {
	c := make(chan struct{}, 1)
	h.mu.Lock()
	h.listeners[c] = struct{}{}
	h.mu.Unlock()
	return c, func() { h.mu.Lock(); delete(h.listeners, c); h.mu.Unlock() }
}
func (h *Hub) Wake() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.listeners {
		select {
		case c <- struct{}{}:
		default:
		}
	}
}
func (h *Hub) JoinNode(id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.nodes[id] {
		return false
	}
	h.nodes[id] = true
	return true
}
func (h *Hub) LeaveNode(id string) { h.mu.Lock(); delete(h.nodes, id); h.mu.Unlock() }
func (h *Hub) AnyNode(ids []string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, id := range ids {
		if h.nodes[id] {
			return true
		}
	}
	return false
}
func (h *Hub) NodeOnline(id string) bool { h.mu.Lock(); defer h.mu.Unlock(); return h.nodes[id] }

package bridge

import (
	"strings"
	"sync"
)

// Wakeups are hints only. SQL is the source of truth; periodic scans cover a
// commit followed by a crash before notification. Memory never holds messages.
type Hub struct {
	mu        sync.Mutex
	listeners map[*Subscription]struct{}
	nodes     map[string]bool
	revision  uint64
}

const (
	TopicPublic   = "chat.public"
	TopicNames    = "players.names"
	TopicAccounts = "accounts.changed"
	TopicIcons    = "launcher.icons"
	TopicNotices  = "notices.changed"
)

func UserTopic(id string) string                         { return "user:" + id }
func ConversationTopic(user, conversation string) string { return UserTopic(user) + "/" + conversation }

// The signal coalesces, while pending topics retain all reasons for the wakeup.
type Subscription struct {
	Signal  chan struct{}
	mu      sync.Mutex
	topics  map[string]bool
	pending map[string]bool
}

func (s *Subscription) Take() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.pending
	s.pending = map[string]bool{}
	return items
}
func (s *Subscription) Request(topic string) {
	s.mu.Lock()
	s.pending[topic] = true
	s.mu.Unlock()
	select {
	case s.Signal <- struct{}{}:
	default:
	}
}

func NewHub() *Hub { return &Hub{listeners: map[*Subscription]struct{}{}, nodes: map[string]bool{}} }
func (h *Hub) Subscribe() (chan struct{}, func()) {
	s, stop := h.Watch()
	return s.Signal, stop
}
func (h *Hub) Watch(topics ...string) (*Subscription, func()) {
	s := &Subscription{Signal: make(chan struct{}, 1), topics: map[string]bool{}, pending: map[string]bool{}}
	for _, topic := range topics {
		s.topics[topic] = true
	}
	h.mu.Lock()
	h.listeners[s] = struct{}{}
	h.mu.Unlock()
	return s, func() { h.mu.Lock(); delete(h.listeners, s); h.mu.Unlock() }
}
func (h *Hub) Wake(topics ...string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(topics) == 0 {
		topics = []string{"*"}
	}
	for _, topic := range topics {
		if topic == "*" || topic == TopicPublic || topic == TopicAccounts {
			h.revision++
			break
		}
	}
	for s := range h.listeners {
		matched := false
		s.mu.Lock()
		for _, topic := range topics {
			accept := topic == "*" || len(s.topics) == 0 || s.topics[topic]
			if !accept {
				for subscribed := range s.topics {
					if strings.HasPrefix(topic, subscribed+"/") {
						accept = true
						break
					}
				}
			}
			if accept {
				s.pending[topic] = true
				matched = true
			}
		}
		s.mu.Unlock()
		if !matched {
			continue
		}
		select {
		case s.Signal <- struct{}{}:
		default:
		}
	}
}
func (h *Hub) Revision() uint64 { h.mu.Lock(); defer h.mu.Unlock(); return h.revision }
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

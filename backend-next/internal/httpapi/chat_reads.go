package httpapi

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

type chatRead struct {
	revision uint64
	ready    chan struct{}
	expires  time.Time
	messages []store.ChatMessage
	err      error
}

// Public messages only. No sessions, private messages or permission decisions
// enter this cache. SQL remains authoritative; Hub revisions invalidate wakeup
// reads and the short TTL covers changes committed without a wakeup.
type sharedChatReads struct {
	mu            sync.Mutex
	entries       map[int64]*chatRead
	reads, shared atomic.Uint64
}

func (c *sharedChatReads) load(ctx context.Context, cursor int64, revision uint64,
	query func(context.Context) ([]store.ChatMessage, error)) ([]store.ChatMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	if old := c.entries[cursor]; old != nil && old.revision == revision {
		select {
		case <-old.ready:
			if old.err == nil && time.Now().Before(old.expires) {
				c.shared.Add(1)
				c.mu.Unlock()
				return old.messages, nil
			}
		default:
			c.shared.Add(1)
			c.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-old.ready:
				return old.messages, old.err
			}
		}
	}
	if c.entries == nil {
		c.entries = make(map[int64]*chatRead)
	}
	// ponytail: bound divergent cursors to 128 cached pages. Slow/reconnecting
	// readers fall back to SQL; add a shared tail buffer only if needed later.
	if len(c.entries) >= 128 {
		for key, entry := range c.entries {
			select {
			case <-entry.ready:
				delete(c.entries, key)
			default:
			}
		}
	}
	if len(c.entries) >= 128 {
		c.mu.Unlock()
		c.reads.Add(1)
		return query(ctx)
	}
	entry := &chatRead{revision: revision, ready: make(chan struct{})}
	c.entries[cursor] = entry
	c.mu.Unlock()
	c.reads.Add(1)
	messages, err := query(ctx)
	c.mu.Lock()
	entry.messages, entry.err, entry.expires = messages, err, time.Now().Add(time.Second)
	close(entry.ready)
	if err != nil && c.entries[cursor] == entry {
		delete(c.entries, cursor)
	}
	c.mu.Unlock()
	return messages, err
}

func (s *Server) publicChatAfter(ctx context.Context, cursor int64) ([]store.ChatMessage, error) {
	return s.chatReads.load(ctx, cursor, s.Hub.Revision(), func(context.Context) ([]store.ChatMessage, error) {
		// One reader disconnecting must not cancel a query shared by others.
		ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
		defer cancel()
		return s.Store.Messages(ctx, cursor, true, 50)
	})
}

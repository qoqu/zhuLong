package bot

import (
	"sync"
	"time"
)

// Deduplicator prevents duplicate message processing using a TTL cache.
type Deduplicator struct {
	seen map[string]time.Time
	ttl  time.Duration
	mu   sync.Mutex
}

// NewDeduplicator creates a new deduplicator with the given TTL.
func NewDeduplicator(ttl time.Duration) *Deduplicator {
	d := &Deduplicator{
		seen: make(map[string]time.Time),
		ttl:  ttl,
	}
	// Start cleanup goroutine
	go d.cleanup()
	return d
}

// IsDuplicate returns true if the message ID has been seen within the TTL.
func (d *Deduplicator) IsDuplicate(messageID string) bool {
	if messageID == "" {
		return false
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.seen[messageID]; exists {
		return true
	}

	d.seen[messageID] = time.Now()
	return false
}

// cleanup removes expired entries periodically.
func (d *Deduplicator) cleanup() {
	ticker := time.NewTicker(d.ttl)
	defer ticker.Stop()

	for range ticker.C {
		d.mu.Lock()
		now := time.Now()
		for id, t := range d.seen {
			if now.Sub(t) > d.ttl {
				delete(d.seen, id)
			}
		}
		d.mu.Unlock()
	}
}

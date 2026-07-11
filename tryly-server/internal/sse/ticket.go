package sse

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// TicketStore issues short-lived, single-use tickets that authorize an SSE
// connection without putting a long-lived JWT in the URL query string
// (which leaks into access logs, proxy logs, and browser history).
//
// Flow: an authenticated client POSTs to mint a ticket, then opens the
// EventSource with ?ticket=<value>. The ticket is consumed on first use and
// expires after ticketTTL.
type TicketStore struct {
	mu      sync.Mutex
	entries map[string]ticketEntry
}

type ticketEntry struct {
	userID    int64
	expiresAt time.Time
}

const ticketTTL = 30 * time.Second

func NewTicketStore() *TicketStore {
	return &TicketStore{entries: make(map[string]ticketEntry)}
}

// Issue mints a new opaque ticket bound to userID. Returns an empty string only
// if the system CSPRNG fails, which the caller should treat as an error.
func (s *TicketStore) Issue(userID int64) string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	ticket := hex.EncodeToString(buf)

	s.mu.Lock()
	s.entries[ticket] = ticketEntry{userID: userID, expiresAt: time.Now().Add(ticketTTL)}
	s.pruneLocked()
	s.mu.Unlock()

	return ticket
}

// Consume validates a ticket and returns the bound userID. It is single-use:
// a valid ticket is deleted on read so it cannot be replayed.
func (s *TicketStore) Consume(ticket string) (int64, bool) {
	if ticket == "" {
		return 0, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.entries[ticket]
	if !ok {
		return 0, false
	}
	delete(s.entries, ticket)
	if time.Now().After(entry.expiresAt) {
		return 0, false
	}
	return entry.userID, true
}

// pruneLocked drops expired entries. Caller must hold s.mu.
func (s *TicketStore) pruneLocked() {
	now := time.Now()
	for k, v := range s.entries {
		if now.After(v.expiresAt) {
			delete(s.entries, k)
		}
	}
}

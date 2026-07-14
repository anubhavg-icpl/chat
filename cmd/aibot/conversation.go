package main

import "sync"

// Conversation is an in-memory, per-user chat history store. Each user's
// history is capped at Limit messages (turns); when the cap is exceeded the
// oldest messages are trimmed. A Limit <= 0 disables trimming (unlimited).
type Conversation struct {
	mu     sync.Mutex
	Limit  int
	stores map[string][]Message
}

// NewConversation creates a Conversation with the given per-user message cap.
func NewConversation(limit int) *Conversation {
	return &Conversation{
		Limit:  limit,
		stores: make(map[string][]Message),
	}
}

// Append adds a message to the user's history and trims the oldest entries so
// that at most Limit messages are retained.
func (c *Conversation) Append(user string, msg Message) {
	c.mu.Lock()
	defer c.mu.Unlock()

	msgs := append(c.stores[user], msg)
	if c.Limit > 0 && len(msgs) > c.Limit {
		msgs = msgs[len(msgs)-c.Limit:]
	}
	c.stores[user] = msgs
}

// Messages returns a copy of the user's history, oldest first. Mutating the
// returned slice does not affect the stored history.
func (c *Conversation) Messages(user string) []Message {
	c.mu.Lock()
	defer c.mu.Unlock()

	src := c.stores[user]
	out := make([]Message, len(src))
	copy(out, src)
	return out
}

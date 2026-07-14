package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConversation_AppendTrimming verifies the per-user history is capped at
// Limit messages and that the oldest entries are dropped first.
func TestConversation_AppendTrimming(t *testing.T) {
	tests := []struct {
		name    string
		limit   int
		appends []Message
		want    []Message
	}{
		{
			name:    "under limit keeps all",
			limit:   3,
			appends: []Message{{Role: "user", Content: "a"}, {Role: "assistant", Content: "b"}},
			want:    []Message{{Role: "user", Content: "a"}, {Role: "assistant", Content: "b"}},
		},
		{
			name:    "exactly at limit keeps all",
			limit:   2,
			appends: []Message{{Role: "user", Content: "a"}, {Role: "assistant", Content: "b"}},
			want:    []Message{{Role: "user", Content: "a"}, {Role: "assistant", Content: "b"}},
		},
		{
			name:    "over limit trims oldest",
			limit:   2,
			appends: []Message{{Role: "user", Content: "a"}, {Role: "assistant", Content: "b"}, {Role: "user", Content: "c"}},
			want:    []Message{{Role: "assistant", Content: "b"}, {Role: "user", Content: "c"}},
		},
		{
			name:    "limit one keeps only newest",
			limit:   1,
			appends: []Message{{Role: "user", Content: "a"}, {Role: "assistant", Content: "b"}},
			want:    []Message{{Role: "assistant", Content: "b"}},
		},
		{
			name:    "limit zero is unlimited",
			limit:   0,
			appends: []Message{{Role: "user", Content: "a"}, {Role: "assistant", Content: "b"}, {Role: "user", Content: "c"}},
			want:    []Message{{Role: "user", Content: "a"}, {Role: "assistant", Content: "b"}, {Role: "user", Content: "c"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := NewConversation(tc.limit)
			for _, m := range tc.appends {
				c.Append("alice", m)
			}
			assert.Equal(t, tc.want, c.Messages("alice"))
		})
	}
}

// TestConversation_PerUserIsolation verifies histories are tracked separately
// per screen name.
func TestConversation_PerUserIsolation(t *testing.T) {
	c := NewConversation(8)
	c.Append("alice", Message{Role: "user", Content: "hi"})
	c.Append("bob", Message{Role: "user", Content: "yo"})

	assert.Equal(t, []Message{{Role: "user", Content: "hi"}}, c.Messages("alice"))
	assert.Equal(t, []Message{{Role: "user", Content: "yo"}}, c.Messages("bob"))
	assert.Empty(t, c.Messages("carol"))
}

// TestConversation_MessagesReturnsCopy verifies that mutating the returned
// slice does not affect the stored history.
func TestConversation_MessagesReturnsCopy(t *testing.T) {
	c := NewConversation(8)
	c.Append("alice", Message{Role: "user", Content: "hi"})

	got := c.Messages("alice")
	got[0] = Message{Role: "user", Content: "mutated"}

	assert.Equal(t, []Message{{Role: "user", Content: "hi"}}, c.Messages("alice"))
}

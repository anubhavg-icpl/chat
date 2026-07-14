package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// sentIM records a single SendIM call.
type sentIM struct {
	to   string
	text string
}

// fakeSender is a Sender that records all SendIM calls. It is safe for
// concurrent use.
type fakeSender struct {
	mu   sync.Mutex
	sent []sentIM
}

func (f *fakeSender) SendIM(to, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, sentIM{to: to, text: text})
	return nil
}

func (f *fakeSender) snapshot() []sentIM {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]sentIM, len(f.sent))
	copy(out, f.sent)
	return out
}

// stubCompleter is a Completer that returns a canned reply or error and
// records the messages it was called with.
type stubCompleter struct {
	mu       sync.Mutex
	got      []Message
	reply    string
	err      error
	recvFunc func([]Message)
}

func (s *stubCompleter) Complete(_ context.Context, messages []Message) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.got = messages
	if s.recvFunc != nil {
		s.recvFunc(messages)
	}
	return s.reply, s.err
}

func newTestBot(completer Completer, sender Sender) *Bot {
	return &Bot{
		conversation: NewConversation(8),
		completer:    completer,
		sender:       sender,
		systemPrompt: "sys-prompt",
		timeout:      5 * time.Second,
		log:          log.New(io.Discard, "", 0),
	}
}

// TestBot_OnIM_RepliesWithContent drives a canned Chat Completions response
// from an httptest server through the real ChatClient and asserts the Bot
// replies with the assistant content.
func TestBot_OnIM_RepliesWithContent(t *testing.T) {
	var lastReq chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&lastReq))
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"Hello there!"}}]}`))
	}))
	defer srv.Close()

	bot := newTestBot(NewChatClient(srv.URL, "k", "m"), &fakeSender{})
	sender := &fakeSender{}
	bot.sender = sender

	bot.OnIM("alice", "hi", false)

	sent := sender.snapshot()
	if assert.Len(t, sent, 1) {
		assert.Equal(t, "alice", sent[0].to)
		assert.Equal(t, "Hello there!", sent[0].text)
	}

	// The request must be led by the system prompt, then the user message.
	assert.Equal(t, "system", lastReq.Messages[0].Role)
	assert.Equal(t, "sys-prompt", lastReq.Messages[0].Content)
	assert.Equal(t, "user", lastReq.Messages[1].Role)
	assert.Equal(t, "hi", lastReq.Messages[1].Content)

	// History should now hold the user message and the assistant reply.
	assert.Equal(t, []Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "Hello there!"},
	}, bot.conversation.Messages("alice"))
}

// TestBot_OnIM_IncludesHistory verifies that a second message carries the
// prior turn in the request.
func TestBot_OnIM_IncludesHistory(t *testing.T) {
	stub := &stubCompleter{reply: "r"}
	bot := newTestBot(stub, &fakeSender{})

	bot.OnIM("alice", "first", false)
	bot.OnIM("alice", "second", false)

	want := []Message{
		{Role: "system", Content: "sys-prompt"},
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "r"},
		{Role: "user", Content: "second"},
	}
	assert.Equal(t, want, stub.got)
}

// TestBot_OnIM_APIError500 verifies that a 500 from the API causes the Bot to
// send an error reply instead of the model content, without panicking.
func TestBot_OnIM_APIError500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`internal server error`))
	}))
	defer srv.Close()

	sender := &fakeSender{}
	bot := newTestBot(NewChatClient(srv.URL, "k", "m"), sender)

	assert.NotPanics(t, func() { bot.OnIM("bob", "hello", false) })

	sent := sender.snapshot()
	if assert.Len(t, sent, 1) {
		assert.Equal(t, "bob", sent[0].to)
		assert.Equal(t, errorMessage, sent[0].text)
	}

	// The error reply is still recorded in history as the assistant turn.
	assert.Equal(t, []Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: errorMessage},
	}, bot.conversation.Messages("bob"))
}

// TestBot_OnIM_CompleterError verifies a non-HTTP completer error is handled
// gracefully with an error reply and no panic.
func TestBot_OnIM_CompleterError(t *testing.T) {
	stub := &stubCompleter{err: errors.New("model offline")}
	sender := &fakeSender{}
	bot := newTestBot(stub, sender)

	assert.NotPanics(t, func() { bot.OnIM("carol", "hey", false) })

	sent := sender.snapshot()
	if assert.Len(t, sent, 1) {
		assert.Equal(t, errorMessage, sent[0].text)
	}
}

// TestBot_OnIM_HistoryTrimsPerUser verifies that per-user history respects the
// configured limit across multiple turns.
func TestBot_OnIM_HistoryTrimsPerUser(t *testing.T) {
	stub := &stubCompleter{reply: "r"}
	bot := &Bot{
		conversation: NewConversation(2), // keep only the two most recent messages
		completer:    stub,
		sender:       &fakeSender{},
		systemPrompt: "sys-prompt",
		timeout:      5 * time.Second,
		log:          log.New(io.Discard, "", 0),
	}

	bot.OnIM("alice", "m1", false) // history: user m1, assistant r
	bot.OnIM("alice", "m2", false) // append user m2 -> trims to [assistant r, user m2]

	// With a limit of 2, the assistant reply from the first turn is retained,
	// but the oldest user message (m1) is trimmed. The request therefore
	// carries the system prompt plus the two surviving history messages.
	assert.Equal(t, []Message{
		{Role: "system", Content: "sys-prompt"},
		{Role: "assistant", Content: "r"},
		{Role: "user", Content: "m2"},
	}, stub.got)
}

// TestBot_OnError verifies the OnError handler does not panic.
func TestBot_OnError(t *testing.T) {
	bot := newTestBot(&stubCompleter{}, &fakeSender{})
	assert.NotPanics(t, func() { bot.OnError("980") })
}

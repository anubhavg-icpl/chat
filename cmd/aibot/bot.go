package main

import (
	"context"
	"log"
	"time"
)

// Sender sends an instant message to a recipient. It is satisfied by
// *toc.Client.
type Sender interface {
	SendIM(to, text string) error
}

// Completer produces an assistant reply for a sequence of chat messages. It is
// satisfied by *ChatClient.
type Completer interface {
	Complete(ctx context.Context, messages []Message) (string, error)
}

// defaultSystemPrompt is used when AIBOT_SYSTEM_PROMPT is unset.
const defaultSystemPrompt = "You are a friendly, helpful assistant chatting over AOL Instant Messenger. Keep your replies short, casual, and conversational."

// errorMessage is sent to a user when the model call fails.
const errorMessage = "Sorry, I hit an error and couldn't generate a reply. Please try again in a moment."

// Bot wires incoming TOC instant messages to an LLM completer and replies via
// a Sender. It implements the toc.Handler interface.
type Bot struct {
	conversation *Conversation
	completer    Completer
	sender       Sender
	systemPrompt string
	timeout      time.Duration
	log          *log.Logger
}

// OnIM implements toc.Handler. It appends the incoming user message to that
// user's history, calls the completer with [system prompt + history], then
// appends the assistant reply and sends it back. On completer or sender
// failure it logs the cause and (for completer errors) sends a short error
// message; it never panics.
func (b *Bot) OnIM(from, text string, auto bool) {
	ctx := context.Background()
	if b.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, b.timeout)
		defer cancel()
	}

	b.conversation.Append(from, Message{Role: "user", Content: text})

	history := b.conversation.Messages(from)
	messages := make([]Message, 0, len(history)+1)
	messages = append(messages, Message{Role: "system", Content: b.systemPrompt})
	messages = append(messages, history...)

	reply, err := b.completer.Complete(ctx, messages)
	if err != nil {
		b.log.Printf("completion failed for %q: %v", from, err)
		reply = errorMessage
	}

	b.conversation.Append(from, Message{Role: "assistant", Content: reply})

	if err := b.sender.SendIM(from, reply); err != nil {
		b.log.Printf("send failed to %q: %v", from, err)
	}
}

// OnError implements toc.Handler.
func (b *Bot) OnError(code string) {
	b.log.Printf("toc error: %s", code)
}

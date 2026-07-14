package toc

import "strings"

// EventType classifies a decoded server-pushed TOC message.
type EventType int

// Recognized event types.
const (
	// EventOther is the zero value for messages the client does not parse into
	// a more specific type. The Raw field still contains the full message.
	EventOther EventType = iota
	// EventIM is an incoming instant message (IM_IN / IM_IN2 / IM_IN_ENC2).
	EventIM
	// EventError is a TOC "ERROR:" message.
	EventError
	// EventSignOn is the "SIGN_ON:<version>" message received after login.
	EventSignOn
	// EventConfig is the "CONFIG:<config>" message carrying the buddy config.
	EventConfig
	// EventNick is the "NICK:<formatted name>" message carrying the display
	// name.
	EventNick
	// EventUpdateBuddy is an "UPDATE_BUDDY:..." presence update.
	EventUpdateBuddy
)

// Event is a parsed TOC protocol message delivered to a handler. Only the
// fields relevant to Type are populated; Raw always holds the full message.
type Event struct {
	// Type is the kind of event.
	Type EventType
	// Raw is the full, unmodified message string.
	Raw string
	// From is the sender screen name for EventIM.
	From string
	// Text is the message body for EventIM.
	Text string
	// Auto reports whether an EventIM was an automatic (away) response.
	Auto bool
	// Code is the text following "ERROR:" for EventError (e.g. "980").
	Code string
	// Name carries the payload of EventSignOn, EventConfig, EventNick, and
	// EventUpdateBuddy.
	Name string
}

// Handler receives decoded TOC protocol events for the common message types.
// Implement this interface and assign an instance to [Options.Handler]. Methods
// are invoked from the [Client.Receive] goroutine and must not block on the
// receiving connection (use a separate goroutine to run long tasks).
type Handler interface {
	// OnIM is called for each incoming instant message. autoResponse is true
	// when the message was an automatic away reply.
	OnIM(from, text string, autoResponse bool)
	// OnError is called for TOC "ERROR:" messages. code is the text after the
	// "ERROR:" prefix.
	OnError(code string)
}

// EventHandler is a function-based handler that receives every server message,
// parsed into an [Event]. Assign it to [Options.OnEvent]. It is invoked before
// the typed [Handler] (if any) and is useful when you want access to the
// [*Client] or to message types without a dedicated Handler method.
type EventHandler func(c *Client, ev Event)

// parseEvent decodes a raw TOC message string into an Event. It handles the
// message formats produced by Open OSCAR Server (see server/toc/cmd_server.go):
//
//	IM_IN:<from>:<auto T/F>:<message>
//	IM_IN2:<from>:<auto T/F>:<whisper T/F>:<message>
//	IM_IN_ENC2:<from>:<auto>:<?>:<T>:<class>:<?>:<L>:<lang>:<message>
//	ERROR:<code>[:<args>]
func parseEvent(msg string) Event {
	ev := Event{Raw: msg, Type: EventOther}
	cmd, rest, found := strings.Cut(msg, ":")
	if !found {
		return ev
	}
	switch cmd {
	case "IM_IN":
		ev.Type = EventIM
		p := splitColon(rest, 3)
		ev.From, ev.Auto, ev.Text = p[0], p[1] == "T", p[2]
	case "IM_IN2":
		ev.Type = EventIM
		p := splitColon(rest, 4)
		ev.From, ev.Auto, ev.Text = p[0], p[1] == "T", p[3]
	case "IM_IN_ENC2":
		ev.Type = EventIM
		p := splitColon(rest, 9)
		ev.From, ev.Auto, ev.Text = p[0], p[1] == "T", p[8]
	case "ERROR":
		ev.Type = EventError
		ev.Code = rest
	case "SIGN_ON":
		ev.Type = EventSignOn
		ev.Name = rest
	case "CONFIG":
		ev.Type = EventConfig
		ev.Name = rest
	case "NICK":
		ev.Type = EventNick
		ev.Name = rest
	case "UPDATE_BUDDY":
		ev.Type = EventUpdateBuddy
		ev.Name = rest
	}
	return ev
}

// splitColon splits s on ":" into at most n fields, left-padded with "" when
// fewer than n fields are present. The final field keeps any remaining colons
// (per the TOC spec, everything after the last expected colon is the message).
func splitColon(s string, n int) []string {
	parts := strings.SplitN(s, ":", n)
	for len(parts) < n {
		parts = append(parts, "")
	}
	return parts
}

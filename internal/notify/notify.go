// Package notify sends short operational alerts (a new order, a
// reconciliation drift finding) to wherever an operator actually watches --
// a chat app, not email, since these are meant to be noticed within
// minutes, not found later in an inbox. One Sender interface, swappable
// backends: Telegram is the first (telegram.go), Slack's incoming-webhook
// shape is close enough to need only a second small implementation, not a
// different abstraction.
package notify

import "context"

// Message is deliberately plain text, not HTML: every backend this package
// targets (Telegram, Slack) renders plain text fine, and formatting would
// have to be backend-specific anyway.
type Message struct {
	Title string
	Body  string
}

// Text joins Title and Body the way a chat message should read: the title
// alone when there is nothing else to say.
func (m Message) Text() string {
	if m.Body == "" {
		return m.Title
	}
	return m.Title + "\n" + m.Body
}

// Sender delivers one Message. Implementations are expected to be
// best-effort from the caller's side: a failed notification must never
// fail the operation that triggered it (an order still succeeded even if
// nobody got pinged about it), so every call site treats Send's error as
// log-worthy, not fatal.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

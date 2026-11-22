package mailer

import (
	"context"
	"sync"
)

// recordingTransport keeps every message instead of delivering it.
type recordingTransport struct {
	mu       sync.Mutex
	messages []Message
	err      error
}

// Send records the message, or fails with err if it is set.
func (r *recordingTransport) Send(_ context.Context, msg Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.err != nil {
		return r.err
	}
	r.messages = append(r.messages, msg)
	return nil
}

// sent returns the messages recorded so far.
func (r *recordingTransport) sent() []Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Message(nil), r.messages...)
}

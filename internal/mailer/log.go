package mailer

import (
	"context"

	"github.com/rs/zerolog"
)

// logTransport "delivers" messages by writing them to the configured logger.
type logTransport struct {
	logger zerolog.Logger
}

// newLogTransport creates a logTransport that writes to the given logger.
func newLogTransport(logger zerolog.Logger) *logTransport {
	return &logTransport{
		logger: logger,
	}
}

// Send logs the rendered message. Recipient, sender and subject are logged at info level,
// while the full plain-text body is logged at debug level.
func (t *logTransport) Send(_ context.Context, msg Message) error {
	event := t.logger.Info().Str("driver", string(DriverLog)).Str("from", msg.From.Address).Str("to", msg.To)

	if len(msg.Cc) > 0 {
		event.Strs("cc", msg.Cc)
	}
	if len(msg.Bcc) > 0 {
		event.Strs("bcc", msg.Bcc)
	}

	if msg.ReplyTo != "" {
		event.Str("reply_to", msg.ReplyTo)
	}
	if msg.MessageID != "" {
		event.Str("message_id", msg.MessageID)
	}
	if !msg.ListUnsubscribe.IsZero() {
		event.Str("list_unsubscribe", msg.ListUnsubscribe.header())
	}
	if len(msg.Headers) > 0 {
		event.Interface("headers", msg.Headers)
	}

	if len(msg.Inline) > 0 {
		names := make([]string, len(msg.Inline))
		for i, image := range msg.Inline {
			names[i] = image.Name
		}
		event.Strs("inline_images", names)
	}

	event.Str("subject", msg.Subject).Int("attachments", len(msg.Attachments)).Msg("mail not sent (log driver)")

	t.logger.Debug().Str("driver", string(DriverLog)).Str("to", msg.To).Str("body", msg.PlainBody).Msg("mail body (log driver)")

	return nil
}

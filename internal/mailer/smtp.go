package mailer

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/wneessen/go-mail"
)

const (
	headerListUnsubscribe     mail.Header = "List-Unsubscribe"
	headerListUnsubscribePost mail.Header = "List-Unsubscribe-Post"
)

// smtpTransport delivers messages over SMTP using the configured mail.Client.
type smtpTransport struct {
	client *mail.Client
}

// newSMTPTransport creates an smtpTransport.
func newSMTPTransport(options Options) (*smtpTransport, error) {
	clientOpts := []mail.Option{
		mail.WithPort(options.SMTPPort),
		mail.WithTLSPortPolicy(mail.TLSOpportunistic),
		mail.WithTimeout(10 * time.Second),
	}
	if options.SMTPUsername != "" {
		clientOpts = append(
			clientOpts,
			mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
			mail.WithUsername(options.SMTPUsername),
			mail.WithPassword(options.SMTPPassword),
		)
	}

	client, err := mail.NewClient(options.SMTPHost, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating mail client: %w", err)
	}

	return &smtpTransport{
		client: client,
	}, nil
}

// Send builds a multipart email (plain-text body with an HTML alternative) and dispatches it over the SMTP connection.
func (t *smtpTransport) Send(ctx context.Context, msg Message) error {
	out := mail.NewMsg()

	var fromErr error
	if msg.From.Name == "" {
		fromErr = out.From(msg.From.Address)
	} else {
		fromErr = out.FromFormat(msg.From.Name, msg.From.Address)
	}
	if fromErr != nil {
		return fmt.Errorf("setting sender: %w", fromErr)
	}

	if err := out.To(msg.To); err != nil {
		return fmt.Errorf("setting recipient: %w", err)
	}

	if len(msg.Cc) > 0 {
		if err := out.Cc(msg.Cc...); err != nil {
			return err
		}
	}
	if len(msg.Bcc) > 0 {
		if err := out.Bcc(msg.Bcc...); err != nil {
			return err
		}
	}

	if msg.ReplyTo != "" {
		if err := out.ReplyTo(msg.ReplyTo); err != nil {
			return fmt.Errorf("setting reply-to: %w", err)
		}
	}

	if msg.MessageID != "" {
		out.SetMessageIDWithValue(msg.MessageID)
	}
	if !msg.ListUnsubscribe.IsZero() {
		out.SetGenHeader(headerListUnsubscribe, msg.ListUnsubscribe.header())
		if msg.ListUnsubscribe.OneClick {
			out.SetGenHeader(headerListUnsubscribePost, "List-Unsubscribe=One-Click")
		}
	}
	for _, name := range slices.Sorted(maps.Keys(msg.Headers)) {
		out.SetGenHeader(mail.Header(name), msg.Headers[name])
	}

	out.Subject(msg.Subject)
	out.SetBodyString(mail.TypeTextPlain, msg.PlainBody)
	out.AddAlternativeString(mail.TypeTextHTML, msg.HTMLBody)

	// The Content-ID header takes the ID in angle brackets (RFC 2392), the HTML refers to it without them
	for _, image := range msg.Inline {
		err := out.EmbedReader(
			image.Name,
			bytes.NewReader(image.Data),
			mail.WithFileContentID("<"+image.ContentID+">"),
			mail.WithFileContentType(mail.ContentType(image.ContentType)),
		)
		if err != nil {
			return fmt.Errorf("embedding %q: %w", image.Name, err)
		}
	}

	for _, attachment := range msg.Attachments {
		if err := out.AttachReader(attachment.Filename, bytes.NewReader(attachment.Data)); err != nil {
			return fmt.Errorf("attaching %q: %w", attachment.Filename, err)
		}
	}

	if err := t.client.DialAndSendWithContext(ctx, out); err != nil {
		return fmt.Errorf("sending email: %w", err)
	}

	return nil
}

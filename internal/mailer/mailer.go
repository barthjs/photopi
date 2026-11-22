// Package mailer provides localized email rendering and delivery behind a driver-agnostic interface,
// with interchangeable log and SMTP backends.
//
// The templates are embedded at build time under templates/: a shared HTML and a shared plain-text layout,
// and one content template per email. A value missing from the data renders as nothing, see i18n.BlankMissing.
//
// Templates embed images with {{ cid "logo.png" }}: the image is sent inline with the email,
// from templates/images/ or from SendOptions.InlineImages, instead of being loaded from a URL,
// which many mail clients block. Only the images the HTML uses are attached.
//
// Options set what every email carries: the sender, a Reply-To address such as a support mailbox next to a no-reply sender,
// custom headers, and the domain of the Message-ID. SendOptions override them per email
// and add the List-Unsubscribe headers mailbox providers expect on notifications.
package mailer

import (
	"context"
	"io/fs"

	"github.com/rs/zerolog"

	"github.com/barthjs/photopi/internal/i18n"
)

// Mailer renders localized emails and delivers them through Transport.
type Mailer struct {
	renderer        *renderer
	images          *imageResolver
	transport       Transport
	from            Address
	replyTo         string
	headers         map[string]string
	messageIDDomain string
}

// SendOptions holds the optional settings of one email.
type SendOptions struct {
	// From replaces the default sender when its address is set.
	From Address
	Cc   []string
	Bcc  []string
	// ReplyTo replaces the default Reply-To address when set.
	ReplyTo     string
	Attachments []Attachment
	// InlineImages are images of this email the HTML references with {{ cid "name" }}.
	// They take precedence over template images of the same name.
	InlineImages []InlineImage
	// ListUnsubscribe adds the List-Unsubscribe headers, see ListUnsubscribe.
	ListUnsubscribe ListUnsubscribe
	// Headers are added to this email and take precedence over Options.Headers.
	Headers map[string]string
}

// New creates a Mailer. It parses the embedded templates and constructs the transport selected by Options.Driver.
func New(logger zerolog.Logger, options Options, bundle *i18n.Bundle) (*Mailer, error) {
	return newMailer(logger, options, bundle, templateFS)
}

// newMailer creates a Mailer that reads its templates from templates, laid out like the embedded ones.
// Tests pass templates of their own.
func newMailer(logger zerolog.Logger, options Options, bundle *i18n.Bundle, templates fs.FS) (*Mailer, error) {
	headers, err := mergeHeaders(options.Headers, nil)
	if err != nil {
		return nil, err
	}
	if err := checkDomain(options.MessageIDDomain); err != nil {
		return nil, err
	}

	domain := contentIDDomain(options.MessageIDDomain, options.FromAddress)

	r, err := newRenderer(bundle, templates, domain)
	if err != nil {
		return nil, err
	}

	transport, err := newTransport(logger, options)
	if err != nil {
		return nil, err
	}

	return &Mailer{
		renderer:  r,
		images:    newImageResolver(domain, templates),
		transport: transport,
		from: Address{
			Name:    options.FromName,
			Address: options.FromAddress,
		},
		replyTo:         options.ReplyTo,
		headers:         headers,
		messageIDDomain: options.MessageIDDomain,
	}, nil
}

// RenderOnly renders the templates and returns the generated subject, plain text, and HTML strings.
func (m *Mailer) RenderOnly(templateName string, data any) (subject, plain, html string, err error) {
	return m.renderer.render(templateName, data)
}

// Send renders the named template with the given data and delivers it through the configured Transport
// as a multipart email (plain-text body with an HTML alternative).
func (m *Mailer) Send(ctx context.Context, to string, templateName string, data any, opts ...SendOptions) error {
	subject, plain, html, err := m.renderer.render(templateName, data)
	if err != nil {
		return err
	}
	return m.SendRaw(ctx, subject, plain, html, to, opts...)
}

// SendRaw bypasses the template engine and delivers a pre-rendered email directly.
func (m *Mailer) SendRaw(ctx context.Context, subject, plain, html string, to string, opts ...SendOptions) error {
	msg, err := m.message(to, subject, plain, html, opts)
	if err != nil {
		return err
	}
	return m.transport.Send(ctx, msg)
}

// message builds the message of one email from the defaults of the Mailer and the optional SendOptions.
func (m *Mailer) message(to, subject, plain, html string, opts []SendOptions) (Message, error) {
	var opt SendOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	headers, err := mergeHeaders(m.headers, opt.Headers)
	if err != nil {
		return Message{}, err
	}
	if err := checkListUnsubscribe(opt.ListUnsubscribe); err != nil {
		return Message{}, err
	}
	inline, err := m.images.resolve(html, opt.InlineImages)
	if err != nil {
		return Message{}, err
	}

	msg := Message{
		From:            m.from,
		To:              to,
		Cc:              opt.Cc,
		Bcc:             opt.Bcc,
		ReplyTo:         m.replyTo,
		Subject:         subject,
		PlainBody:       plain,
		HTMLBody:        html,
		Attachments:     opt.Attachments,
		Inline:          inline,
		ListUnsubscribe: opt.ListUnsubscribe,
		Headers:         headers,
	}
	if opt.From.Address != "" {
		msg.From = opt.From
	}
	if opt.ReplyTo != "" {
		msg.ReplyTo = opt.ReplyTo
	}
	if m.messageIDDomain != "" {
		msg.MessageID, err = newMessageID(m.messageIDDomain)
		if err != nil {
			return Message{}, err
		}
	}
	return msg, nil
}

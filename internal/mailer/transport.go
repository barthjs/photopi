package mailer

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
)

// Driver selects the transport that delivers the emails.
type Driver string

const (
	DriverLog  Driver = "log"
	DriverSMTP Driver = "smtp"
)

// Address is the name and email of a sender.
type Address struct {
	Name    string
	Address string
}

// Attachment is a file attached to an outgoing email.
type Attachment struct {
	Filename string
	Data     []byte
}

// Message is a fully rendered email handed to a Transport for delivery.
type Message struct {
	From        Address
	To          string
	Cc          []string
	Bcc         []string
	ReplyTo     string
	Subject     string
	PlainBody   string
	HTMLBody    string
	Attachments []Attachment
	// Inline holds the images the HTML body references as "cid:" URLs.
	Inline []EmbeddedImage
	// MessageID is the Message-ID header without angle brackets. Empty lets the transport generate one.
	MessageID string
	// ListUnsubscribe adds the unsubscribe headers when set.
	ListUnsubscribe ListUnsubscribe
	// Headers holds further headers, already checked by the Mailer.
	Headers map[string]string
}

// ListUnsubscribe tells mail clients how a recipient unsubscribes from a kind of email (RFC 2369).
// Mailbox providers expect it on bulk email such as notifications and newsletters.
type ListUnsubscribe struct {
	// Mailto is the address an unsubscribe request is mailed to.
	Mailto string
	// URL is the https address that unsubscribes the recipient.
	URL string
	// OneClick adds "List-Unsubscribe-Post: List-Unsubscribe=One-Click", so a mail client unsubscribes
	// with a single POST to URL without opening a page (RFC 8058). It requires URL.
	OneClick bool
}

// IsZero reports whether no unsubscribe method is set.
func (l ListUnsubscribe) IsZero() bool {
	return l.Mailto == "" && l.URL == ""
}

// header returns the value of the List-Unsubscribe header, such as "<mailto:a@b>, <https://b/u>".
func (l ListUnsubscribe) header() string {
	var parts []string
	if l.Mailto != "" {
		parts = append(parts, "<mailto:"+l.Mailto+">")
	}
	if l.URL != "" {
		parts = append(parts, "<"+l.URL+">")
	}
	return strings.Join(parts, ", ")
}

// Transport delivers rendered messages.
type Transport interface {
	// Send delivers a single rendered message.
	Send(ctx context.Context, msg Message) error
}

// Options holds the settings of a Mailer and its Transport.
type Options struct {
	Driver      Driver
	FromAddress string
	FromName    string
	// ReplyTo is the Reply-To address of every email unless SendOptions sets one,
	// for example a support address next to a no-reply sender. Empty sends no Reply-To.
	ReplyTo string
	// Headers are added to every email. SendOptions.Headers take precedence for the same name.
	// Headers with a dedicated field, such as From or List-Unsubscribe, are rejected.
	Headers map[string]string
	// MessageIDDomain is the domain of the Message-ID of every email ("<random@domain>"),
	// usually the sending domain, so the IDs match the sender. Empty lets the transport generate the ID.
	MessageIDDomain string
	SMTPHost        string
	SMTPPort        int
	SMTPUsername    string
	SMTPPassword    string
	// Transport replaces the transport selected by Driver, for example with a fake that records messages in tests.
	// Nil selects the transport by Driver.
	Transport Transport
}

// newTransport creates a Transport for the configured Driver.
func newTransport(logger zerolog.Logger, options Options) (Transport, error) {
	if options.Transport != nil {
		return options.Transport, nil
	}

	switch options.Driver {
	case DriverLog:
		return newLogTransport(logger), nil
	case DriverSMTP:
		return newSMTPTransport(options)
	default:
		return nil, fmt.Errorf("mailer: unknown driver %q", options.Driver)
	}
}

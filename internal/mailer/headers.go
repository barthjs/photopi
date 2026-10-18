package mailer

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/textproto"
	"net/url"
	"strings"
)

// ErrInvalidHeader is returned when a header, an unsubscribe setting, or a Message-ID domain cannot be sent.
var ErrInvalidHeader = errors.New("mailer: invalid header")

// reservedHeaders have dedicated fields or are written by the transport, so custom headers must not set them.
var reservedHeaders = map[string]bool{
	"From": true, "To": true, "Cc": true, "Bcc": true, "Reply-To": true, "Subject": true,
	"Date": true, "Message-Id": true, "Mime-Version": true, "Content-Type": true,
	"Content-Transfer-Encoding": true, "List-Unsubscribe": true, "List-Unsubscribe-Post": true,
}

// mergeHeaders checks the global and per-email headers and returns them combined,
// with the per-email value taking precedence for the same name. Names are compared case-insensitively.
func mergeHeaders(global, perEmail map[string]string) (map[string]string, error) {
	if len(global) == 0 && len(perEmail) == 0 {
		return nil, nil
	}

	merged := make(map[string]string, len(global)+len(perEmail))
	for _, headers := range []map[string]string{global, perEmail} {
		for name, value := range headers {
			canonical, err := checkHeader(name, value)
			if err != nil {
				return nil, err
			}
			merged[canonical] = value
		}
	}
	return merged, nil
}

// checkHeader returns the canonical form of a header name after checking that the header can be sent as it is:
// a field name of printable ASCII without colon, a value without line breaks, and no header that has a dedicated field.
func checkHeader(name, value string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("%w: empty name", ErrInvalidHeader)
	}
	for _, r := range name {
		if r <= ' ' || r > '~' || r == ':' {
			return "", fmt.Errorf("%w: name %q", ErrInvalidHeader, name)
		}
	}

	canonical := textproto.CanonicalMIMEHeaderKey(name)
	if reservedHeaders[canonical] {
		return "", fmt.Errorf("%w: %s has a dedicated option", ErrInvalidHeader, canonical)
	}
	if strings.ContainsAny(value, "\r\n") {
		return "", fmt.Errorf("%w: value of %s contains a line break", ErrInvalidHeader, canonical)
	}
	return canonical, nil
}

// checkListUnsubscribe checks that the unsubscribe targets are a plain address and a https URL,
// as mailbox providers only follow those, and that one-click unsubscribing has a URL to post to.
func checkListUnsubscribe(l ListUnsubscribe) error {
	if strings.ContainsAny(l.Mailto, "\r\n<>, ") {
		return fmt.Errorf("%w: unsubscribe address %q", ErrInvalidHeader, l.Mailto)
	}
	if l.URL != "" {
		parsed, err := url.Parse(l.URL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || strings.ContainsAny(l.URL, "\r\n<>, ") {
			return fmt.Errorf("%w: unsubscribe URL %q must be an https URL", ErrInvalidHeader, l.URL)
		}
	}
	if l.OneClick && l.URL == "" {
		return fmt.Errorf("%w: one-click unsubscribing requires a URL", ErrInvalidHeader)
	}
	return nil
}

// checkDomain checks that a Message-ID domain is a bare host name.
func checkDomain(domain string) error {
	if strings.ContainsAny(domain, "\r\n<>@ ") {
		return fmt.Errorf("%w: Message-ID domain %q", ErrInvalidHeader, domain)
	}
	return nil
}

// newMessageID returns a unique Message-ID for the domain, without angle brackets.
func newMessageID(domain string) (string, error) {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generating message id: %w", err)
	}
	return hex.EncodeToString(random) + "@" + domain, nil
}

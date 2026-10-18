package mailer

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSMTPTransport(t *testing.T) {
	t.Parallel()

	t.Run("rejects an invalid host", func(t *testing.T) {
		t.Parallel()

		_, err := newSMTPTransport(Options{
			Driver:   DriverSMTP,
			SMTPPort: 25,
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "creating mail client")
	})
}

func TestSMTPTransport_Send(t *testing.T) {
	t.Parallel()

	t.Run("sends the message", func(t *testing.T) {
		t.Parallel()

		server := newFakeSMTPServer(t)
		transport := newTestSMTPTransport(t, server.port, "", "")

		err := transport.Send(t.Context(), testMessage())

		require.NoError(t, err)
		messages := server.messages()
		require.Len(t, messages, 1)
		received := messages[0]
		assert.Equal(t, "sender@example.test", received.From)
		assert.ElementsMatch(t, []string{"to@example.test", "cc@example.test", "bcc@example.test"}, received.Recipients)
		assert.Contains(t, received.Data, "Subject: Subject line")
		assert.Contains(t, received.Data, `From: "Sender" <sender@example.test>`)
		assert.Contains(t, received.Data, "Reply-To: <reply@example.test>")
		assert.Contains(t, received.Data, "Cc: <cc@example.test>")
		assert.NotContains(t, received.Data, "bcc@example.test", "blind copies stay out of the headers")
		assert.Contains(t, received.Data, "Plain body")
		assert.Contains(t, received.Data, "text/html")
		assert.Contains(t, received.Data, `filename="note.txt"`)
		assert.Contains(t, received.Data, base64.StdEncoding.EncodeToString([]byte("attachment content")))
		assert.Empty(t, received.Username, "no authentication without a username")
	})

	t.Run("sends a sender without name", func(t *testing.T) {
		t.Parallel()

		server := newFakeSMTPServer(t)
		transport := newTestSMTPTransport(t, server.port, "", "")
		msg := Message{
			From: Address{
				Address: "sender@example.test",
			},
			To:        "to@example.test",
			Subject:   "s",
			PlainBody: "p",
			HTMLBody:  "h",
		}

		err := transport.Send(t.Context(), msg)

		require.NoError(t, err)
		messages := server.messages()
		require.Len(t, messages, 1)
		assert.Contains(t, messages[0].Data, "From: <sender@example.test>")
		assert.NotContains(t, messages[0].Data, "Reply-To")
	})

	t.Run("sends the custom headers", func(t *testing.T) {
		t.Parallel()

		server := newFakeSMTPServer(t)
		transport := newTestSMTPTransport(t, server.port, "", "")
		msg := testMessage()
		msg.MessageID = "0123456789abcdef@example.test"
		msg.ListUnsubscribe = ListUnsubscribe{
			Mailto:   "unsubscribe@example.test",
			URL:      "https://example.test/u/1",
			OneClick: true,
		}
		msg.Headers = map[string]string{"X-Campaign": "spring", "X-Mailer": "app"}

		err := transport.Send(t.Context(), msg)

		require.NoError(t, err)
		messages := server.messages()
		require.Len(t, messages, 1)
		// Long header lines are folded at a comma (RFC 5322), so they are unfolded before comparing
		data := strings.ReplaceAll(messages[0].Data, "\r\n ", " ")
		assert.Contains(t, data, "Message-ID: <0123456789abcdef@example.test>")
		assert.Contains(t, data, "List-Unsubscribe: <mailto:unsubscribe@example.test>, <https://example.test/u/1>")
		assert.Contains(t, data, "List-Unsubscribe-Post: List-Unsubscribe=One-Click")
		assert.Contains(t, data, "X-Campaign: spring")
		assert.Contains(t, data, "X-Mailer: app")
	})

	t.Run("sends the unsubscribe address without one-click", func(t *testing.T) {
		t.Parallel()

		server := newFakeSMTPServer(t)
		transport := newTestSMTPTransport(t, server.port, "", "")
		msg := testMessage()
		msg.ListUnsubscribe = ListUnsubscribe{
			Mailto: "unsubscribe@example.test",
		}

		err := transport.Send(t.Context(), msg)

		require.NoError(t, err)
		data := server.messages()[0].Data
		assert.Contains(t, data, "List-Unsubscribe: <mailto:unsubscribe@example.test>")
		assert.NotContains(t, data, "List-Unsubscribe-Post")
	})

	t.Run("embeds the inline images", func(t *testing.T) {
		t.Parallel()

		server := newFakeSMTPServer(t)
		transport := newTestSMTPTransport(t, server.port, "", "")
		msg := testMessage()
		msg.HTMLBody = `<img src="cid:logo.png@example.test">`
		msg.Inline = []EmbeddedImage{
			{
				Name:        "logo.png",
				ContentID:   "logo.png@example.test",
				ContentType: "image/png",
				Data:        testPNG(t),
			},
		}

		err := transport.Send(t.Context(), msg)

		require.NoError(t, err)
		messages := server.messages()
		require.Len(t, messages, 1)
		data := messages[0].Data
		assert.Contains(t, data, "multipart/related")
		assert.Contains(t, data, "Content-Id: <logo.png@example.test>")
		assert.Contains(t, data, "Content-Type: image/png")
		assert.Contains(t, data, "Content-Disposition: inline")
	})

	t.Run("authenticates with the credentials", func(t *testing.T) {
		t.Parallel()

		server := newFakeSMTPServer(t)
		server.password = "secret"
		transport := newTestSMTPTransport(t, server.port, "user", "secret")

		err := transport.Send(t.Context(), testMessage())

		require.NoError(t, err)
		messages := server.messages()
		require.Len(t, messages, 1)
		assert.Equal(t, "user", messages[0].Username)
	})

	tests := []struct {
		name    string
		mutate  func(*Message)
		wantErr string
	}{
		{
			name: "rejects an invalid sender",
			mutate: func(m *Message) {
				m.From = Address{
					Address: "not an address",
				}
			},
			wantErr: "setting sender",
		},
		{
			name: "rejects an invalid named sender",
			mutate: func(m *Message) {
				m.From = Address{
					Name:    "Name",
					Address: "not an address",
				}
			},
			wantErr: "setting sender",
		},
		{
			name:    "rejects an invalid recipient",
			mutate:  func(m *Message) { m.To = "not an address" },
			wantErr: "setting recipient",
		},
		{
			name:   "rejects an invalid copy",
			mutate: func(m *Message) { m.Cc = []string{"not an address"} },
		},
		{
			name:   "rejects an invalid blind copy",
			mutate: func(m *Message) { m.Bcc = []string{"not an address"} },
		},
		{
			name:    "rejects an invalid reply address",
			mutate:  func(m *Message) { m.ReplyTo = "not an address" },
			wantErr: "setting reply-to",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := newFakeSMTPServer(t)
			transport := newTestSMTPTransport(t, server.port, "", "")
			msg := testMessage()
			tt.mutate(&msg)

			err := transport.Send(t.Context(), msg)

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
			assert.Empty(t, server.messages())
		})
	}

	t.Run("rejects a wrong password", func(t *testing.T) {
		t.Parallel()

		server := newFakeSMTPServer(t)
		server.password = "secret"
		transport := newTestSMTPTransport(t, server.port, "user", "wrong")

		err := transport.Send(t.Context(), testMessage())

		require.Error(t, err)
		assert.Empty(t, server.messages())
	})

	t.Run("reports a rejected recipient", func(t *testing.T) {
		t.Parallel()

		server := newFakeSMTPServer(t)
		server.rejectRecipient = "to@example.test"
		transport := newTestSMTPTransport(t, server.port, "", "")

		err := transport.Send(t.Context(), testMessage())

		require.Error(t, err)
		assert.ErrorContains(t, err, "sending email")
	})

	t.Run("reports an unreachable server", func(t *testing.T) {
		t.Parallel()

		transport := newTestSMTPTransport(t, unusedPort(t), "", "")

		err := transport.Send(t.Context(), testMessage())

		require.Error(t, err)
		assert.ErrorContains(t, err, "sending email")
	})

	t.Run("reports a canceled context", func(t *testing.T) {
		t.Parallel()

		server := newFakeSMTPServer(t)
		transport := newTestSMTPTransport(t, server.port, "", "")
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := transport.Send(ctx, testMessage())

		require.Error(t, err)
		assert.Empty(t, server.messages())
	})
}

// newTestSMTPTransport creates an SMTP transport for the fake server on port.
func newTestSMTPTransport(t *testing.T, port int, username, password string) *smtpTransport {
	t.Helper()

	transport, err := newSMTPTransport(Options{
		Driver:       DriverSMTP,
		SMTPHost:     "127.0.0.1",
		SMTPPort:     port,
		SMTPUsername: username,
		SMTPPassword: password,
	})
	require.NoError(t, err)
	return transport
}

// testMessage returns a message that uses every field the SMTP transport sends.
func testMessage() Message {
	return Message{
		From: Address{
			Name:    "Sender",
			Address: "sender@example.test",
		},
		To:        "to@example.test",
		Cc:        []string{"cc@example.test"},
		Bcc:       []string{"bcc@example.test"},
		ReplyTo:   "reply@example.test",
		Subject:   "Subject line",
		PlainBody: "Plain body",
		HTMLBody:  "<p>HTML body</p>",
		Attachments: []Attachment{
			{
				Filename: "note.txt",
				Data:     []byte("attachment content"),
			},
		},
	}
}

package mailer

import (
	"bytes"
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
)

func TestLogTransport_Send(t *testing.T) {
	t.Parallel()

	t.Run("logs the message", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		var buf bytes.Buffer
		transport := newLogTransport(zerolog.New(&buf).Level(zerolog.DebugLevel))
		msg := Message{
			From: Address{
				Address: f.Email(),
			},
			To:        f.Email(),
			Cc:        []string{f.Email()},
			Bcc:       []string{f.Email()},
			ReplyTo:   f.Email(),
			Subject:   f.LoremIpsumSentence(6),
			PlainBody: f.LoremIpsumParagraph(1, 3, 8, " "),
			Attachments: []Attachment{
				{
					Filename: f.LetterN(8) + ".txt",
				},
			},
			Inline: []EmbeddedImage{
				{
					Name: f.LetterN(8) + ".png",
				},
				{
					Name: f.LetterN(8) + ".jpg",
				},
			},
			MessageID: f.LetterN(16) + "@" + f.DomainName(),
			ListUnsubscribe: ListUnsubscribe{
				Mailto: f.Email(),
			},
			Headers: map[string]string{"X-Campaign": f.Word()},
		}

		err := transport.Send(t.Context(), msg)

		require.NoError(t, err)
		entries := logEntries(t, &buf)
		require.Len(t, entries, 2)
		info := entries[0]
		assert.Equal(t, "info", info["level"])
		assert.Equal(t, "log", info["driver"])
		assert.Equal(t, msg.From.Address, info["from"])
		assert.Equal(t, msg.To, info["to"])
		assert.Equal(t, []any{msg.Cc[0]}, info["cc"])
		assert.Equal(t, []any{msg.Bcc[0]}, info["bcc"])
		assert.Equal(t, msg.ReplyTo, info["reply_to"])
		assert.Equal(t, msg.Subject, info["subject"])
		assert.InDelta(t, 1, info["attachments"], 0)
		assert.Equal(t, msg.MessageID, info["message_id"])
		assert.Equal(t, "<mailto:"+msg.ListUnsubscribe.Mailto+">", info["list_unsubscribe"])
		assert.Equal(t, map[string]any{"X-Campaign": msg.Headers["X-Campaign"]}, info["headers"])
		assert.Equal(t, []any{msg.Inline[0].Name, msg.Inline[1].Name}, info["inline_images"])
		assert.NotContains(t, info, "body", "the body is only logged at debug level")
		debug := entries[1]
		assert.Equal(t, "debug", debug["level"])
		assert.Equal(t, msg.PlainBody, debug["body"])
	})

	t.Run("omits empty fields", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		transport := newLogTransport(zerolog.New(&buf).Level(zerolog.InfoLevel))

		err := transport.Send(t.Context(), Message{
			To:      "to@example.test",
			Subject: "s",
		})

		require.NoError(t, err)
		entries := logEntries(t, &buf)
		require.Len(t, entries, 1, "the body stays out of the log above debug level")
		for _, field := range []string{"cc", "bcc", "reply_to", "message_id", "list_unsubscribe", "headers"} {
			assert.NotContains(t, entries[0], field)
		}
	})
}

// logEntries decodes the JSON lines of a log buffer.
func logEntries(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()

	var entries []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(buf.String()), "\n") {
		var entry map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		entries = append(entries, entry)
	}
	return entries
}

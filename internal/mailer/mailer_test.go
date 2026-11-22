package mailer

import (
	"errors"
	"io/fs"
	"maps"
	"testing"
	"testing/fstest"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
	"github.com/barthjs/photopi/internal/i18n"
)

func TestNew(t *testing.T) {
	t.Parallel()

	bundle, err := i18n.New()
	require.NoError(t, err)

	t.Run("builds the transport of the driver", func(t *testing.T) {
		t.Parallel()

		m, err := New(zerolog.Nop(), Options{
			Driver:      DriverLog,
			FromName:    "App",
			FromAddress: "app@example.test",
		}, bundle)

		require.NoError(t, err)
		assert.IsType(t, &logTransport{}, m.transport)
		assert.Equal(t, Address{
			Name:    "App",
			Address: "app@example.test",
		}, m.from)
	})

	t.Run("rejects an invalid default header", func(t *testing.T) {
		t.Parallel()

		_, err := New(zerolog.Nop(), Options{
			Driver:  DriverLog,
			Headers: map[string]string{"Subject": "x"},
		}, bundle)

		require.ErrorIs(t, err, ErrInvalidHeader)
	})

	t.Run("rejects an invalid message id domain", func(t *testing.T) {
		t.Parallel()

		_, err := New(zerolog.Nop(), Options{
			Driver:          DriverLog,
			MessageIDDomain: "no@domain",
		}, bundle)

		require.ErrorIs(t, err, ErrInvalidHeader)
	})

	t.Run("rejects an invalid template", func(t *testing.T) {
		t.Parallel()

		templates := withEmbeddedTemplates(t, fstest.MapFS{"templates/broken.gohtml": {
			Data: []byte(`{{ define "subject" }`),
		}})

		_, err := newMailer(zerolog.Nop(), Options{
			Driver: DriverLog,
		}, bundle, templates)

		require.Error(t, err)
	})

	t.Run("rejects an unknown driver", func(t *testing.T) {
		t.Parallel()

		_, err := New(zerolog.Nop(), Options{
			Driver: "pigeon",
		}, bundle)

		require.Error(t, err)
	})
}

func TestMailer_RenderOnly(t *testing.T) {
	t.Parallel()

	t.Run("renders the template without sending", func(t *testing.T) {
		t.Parallel()

		m, transport := newTestMailer(t)
		name := apitest.Faker(t).FirstName()

		subject, plain, html, err := m.RenderOnly("welcome.gohtml", renderData{
			Locale: "de",
			Name:   name,
		})

		require.NoError(t, err)
		assert.Equal(t, "Hallo "+name, subject)
		assert.Contains(t, plain, "Guten Tag "+name)
		assert.Contains(t, html, "<p>Guten Tag "+name+"</p>")
		assert.Empty(t, transport.sent(), "rendering sends nothing")
	})
}

func TestMailer_Send(t *testing.T) {
	t.Parallel()

	t.Run("sends the rendered email", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		m, transport := newTestMailer(t)
		attachment := Attachment{
			Filename: f.LetterN(8) + ".txt",
			Data:     []byte(f.LoremIpsumSentence(6)),
		}
		name := f.FirstName()
		to := f.Email()
		cc := f.Email()
		bcc := f.Email()
		replyTo := f.Email()

		err := m.Send(t.Context(), to, "welcome.gohtml", renderData{
			Locale: "en",
			Name:   name,
		}, SendOptions{
			Cc:          []string{cc},
			Bcc:         []string{bcc},
			ReplyTo:     replyTo,
			Attachments: []Attachment{attachment},
		})

		require.NoError(t, err)
		sent := transport.sent()
		require.Len(t, sent, 1)
		msg := sent[0]
		assert.Equal(t, m.from, msg.From)
		assert.Equal(t, to, msg.To)
		assert.Equal(t, "Hi "+name, msg.Subject)
		assert.Contains(t, msg.PlainBody, "Hello "+name)
		assert.Contains(t, msg.HTMLBody, "<p>Hello "+name+"</p>")
		assert.Equal(t, []string{cc}, msg.Cc)
		assert.Equal(t, []string{bcc}, msg.Bcc)
		assert.Equal(t, replyTo, msg.ReplyTo)
		assert.Equal(t, []Attachment{attachment}, msg.Attachments)
	})

	t.Run("embeds the inline images", func(t *testing.T) {
		t.Parallel()

		m, transport, logo := newInlineMailer(t)
		photo := InlineImage{
			Name: "photo.png",
			Data: logo,
		}

		err := m.Send(t.Context(), "to@example.test", "inline.gohtml", inlineData(), SendOptions{
			InlineImages: []InlineImage{photo},
		})

		require.NoError(t, err)
		sent := transport.sent()
		require.Len(t, sent, 1)
		require.Len(t, sent[0].Inline, 2, "only the images the HTML uses")
		assert.Equal(t, "logo.png@example.test", sent[0].Inline[0].ContentID)
		assert.Equal(t, "photo.png@example.test", sent[0].Inline[1].ContentID)
		assert.Contains(t, sent[0].HTMLBody, `src="cid:logo.png@example.test"`)
	})

	t.Run("rejects invalid options", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			options SendOptions
		}{
			{
				name: "rejects a header with a dedicated field",
				options: SendOptions{
					Headers: map[string]string{"Bcc": "evil@example.test"},
				},
			},
			{
				name: "rejects a line break in a header value",
				options: SendOptions{
					Headers: map[string]string{"X-Note": "a\r\nb"},
				},
			},
			{
				name: "rejects one-click without a URL",
				options: SendOptions{
					ListUnsubscribe: ListUnsubscribe{
						Mailto:   "u@example.test",
						OneClick: true,
					},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				m, transport := newTestMailer(t)

				err := m.Send(t.Context(), "to@example.test", "welcome.gohtml", renderData{
					Locale: "en",
				}, tt.options)

				require.ErrorIs(t, err, ErrInvalidHeader)
				assert.Empty(t, transport.sent())
			})
		}
	})

	t.Run("rejects an unknown template", func(t *testing.T) {
		t.Parallel()

		m, transport := newTestMailer(t)

		err := m.Send(t.Context(), "to@example.test", "missing.gohtml", nil)

		require.Error(t, err)
		assert.Empty(t, transport.sent())
	})

	t.Run("rejects an image that is neither a template image nor part of the email", func(t *testing.T) {
		t.Parallel()

		m, _, _ := newInlineMailer(t)

		err := m.Send(t.Context(), "to@example.test", "inline.gohtml", inlineData())

		assert.ErrorIs(t, err, ErrImageNotFound, "the photo is neither a template image nor part of the email")
	})

	t.Run("reports a transport failure", func(t *testing.T) {
		t.Parallel()

		m, transport := newTestMailer(t)
		transport.err = errors.New("delivery failed")

		err := m.Send(t.Context(), "to@example.test", "welcome.gohtml", renderData{
			Locale: "en",
		})

		assert.ErrorIs(t, err, transport.err)
	})
}

func TestMailer_SendRaw(t *testing.T) {
	t.Parallel()

	t.Run("sends the email", func(t *testing.T) {
		t.Parallel()

		m, transport := newTestMailer(t)

		err := m.SendRaw(t.Context(), "Subject", "Plain", "<p>HTML</p>", "to@example.test", SendOptions{
			ReplyTo: "reply@example.test",
		})

		require.NoError(t, err)
		sent := transport.sent()
		require.Len(t, sent, 1)
		assert.Equal(t, Message{
			From: Address{
				Name:    "App",
				Address: "app@example.test",
			},
			To:        "to@example.test",
			ReplyTo:   "reply@example.test",
			Subject:   "Subject",
			PlainBody: "Plain",
			HTMLBody:  "<p>HTML</p>",
		}, sent[0])
	})

	t.Run("applies the defaults", func(t *testing.T) {
		t.Parallel()

		m, transport := newDefaultsMailer(t)

		err := m.SendRaw(t.Context(), "Subject", "Plain", "<p>HTML</p>", "to@example.test")

		require.NoError(t, err)
		sent := transport.sent()
		require.Len(t, sent, 1)
		defaults := sent[0]
		assert.Equal(t, Address{
			Name:    "App",
			Address: "no-reply@example.test",
		}, defaults.From)
		assert.Equal(t, "support@example.test", defaults.ReplyTo)
		assert.Equal(t, map[string]string{"X-Mailer": "app", "X-Campaign": "default"}, defaults.Headers)
		assert.Regexp(t, `@example\.test$`, defaults.MessageID)
		assert.True(t, defaults.ListUnsubscribe.IsZero())
	})

	t.Run("prefers the options over the defaults", func(t *testing.T) {
		t.Parallel()

		m, transport := newDefaultsMailer(t)

		err := m.SendRaw(t.Context(), "Subject", "Plain", "<p>HTML</p>", "to@example.test", SendOptions{
			From: Address{
				Name:    "Team",
				Address: "team@example.test",
			},
			ReplyTo: "other@example.test",
			Headers: map[string]string{"X-Campaign": "summer"},
			ListUnsubscribe: ListUnsubscribe{
				URL:      "https://example.test/u",
				OneClick: true,
			},
		})

		require.NoError(t, err)
		sent := transport.sent()
		require.Len(t, sent, 1)
		overridden := sent[0]
		assert.Equal(t, Address{
			Name:    "Team",
			Address: "team@example.test",
		}, overridden.From)
		assert.Equal(t, "other@example.test", overridden.ReplyTo)
		assert.Equal(t, map[string]string{"X-Mailer": "app", "X-Campaign": "summer"}, overridden.Headers)
		assert.Equal(t, ListUnsubscribe{
			URL:      "https://example.test/u",
			OneClick: true,
		}, overridden.ListUnsubscribe)
	})

	t.Run("gives every email its own message id", func(t *testing.T) {
		t.Parallel()

		m, transport := newDefaultsMailer(t)
		require.NoError(t, m.SendRaw(t.Context(), "Subject", "Plain", "<p>HTML</p>", "to@example.test"))

		err := m.SendRaw(t.Context(), "Subject", "Plain", "<p>HTML</p>", "to@example.test")

		require.NoError(t, err)
		sent := transport.sent()
		require.Len(t, sent, 2)
		assert.NotEqual(t, sent[0].MessageID, sent[1].MessageID, "every email gets its own ID")
	})

	t.Run("embeds the referenced images", func(t *testing.T) {
		t.Parallel()

		m, transport := newTestMailer(t)
		photo := InlineImage{
			Name: "photo.png",
			Data: testPNG(t),
		}

		err := m.SendRaw(t.Context(), "Subject", "Plain", `<img src="cid:photo.png@example.test">`, "to@example.test",
			SendOptions{
				InlineImages: []InlineImage{photo},
			})

		require.NoError(t, err)
		sent := transport.sent()
		require.Len(t, sent, 1)
		require.Len(t, sent[0].Inline, 1)
		assert.Equal(t, "image/png", sent[0].Inline[0].ContentType)
	})
}

// newTestMailer creates a mailer on the test templates that records instead of delivering.
func newTestMailer(t *testing.T) (*Mailer, *recordingTransport) {
	t.Helper()

	transport := &recordingTransport{}
	return &Mailer{
		renderer:  newTestRenderer(t),
		images:    newImageResolver("example.test", testTemplates(testContent)),
		transport: transport,
		from: Address{
			Name:    "App",
			Address: "app@example.test",
		},
	}, transport
}

// newInlineMailer creates a recording mailer whose template embeds a template image and an image of the email,
// and returns the PNG data it uses for both.
func newInlineMailer(t *testing.T) (*Mailer, *recordingTransport, []byte) {
	t.Helper()

	bundle, err := i18n.New()
	require.NoError(t, err)
	logo := testPNG(t)
	transport := &recordingTransport{}
	templates := withEmbeddedTemplates(t, fstest.MapFS{
		"templates/images/logo.png": {
			Data: logo,
		},
		"templates/images/unused.png": {
			Data: logo,
		},
		"templates/inline.gohtml": {
			Data: []byte(`{{ define "subject" }}Inline{{ end }}` +
				`{{ define "content_html" }}<img src="{{ cid "logo.png" }}"><img src="{{ cid "photo.png" }}">{{ end }}` +
				`{{ define "content_plain" }}Body{{ end }}`),
		},
	})
	m, err := newMailer(zerolog.Nop(), Options{
		Driver:      DriverLog,
		FromAddress: "app@example.test",
		Transport:   transport,
	}, bundle, templates)
	require.NoError(t, err)
	return m, transport, logo
}

// withEmbeddedTemplates returns the embedded templates with the files of extra added or replaced.
func withEmbeddedTemplates(t *testing.T, extra fstest.MapFS) fstest.MapFS {
	t.Helper()

	templates := fstest.MapFS{}
	err := fs.WalkDir(templateFS, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := fs.ReadFile(templateFS, path)
		if err != nil {
			return err
		}
		templates[path] = &fstest.MapFile{
			Data: data,
		}
		return nil
	})
	require.NoError(t, err)

	maps.Copy(templates, extra)
	return templates
}

// inlineData returns the data the inline image template renders.
func inlineData() map[string]any {
	return map[string]any{"Locale": "en", "AppName": "App"}
}

// newDefaultsMailer creates a recording mailer with a default sender, reply address, headers, and message id domain.
func newDefaultsMailer(t *testing.T) (*Mailer, *recordingTransport) {
	t.Helper()

	bundle, err := i18n.New()
	require.NoError(t, err)
	transport := &recordingTransport{}
	m, err := New(zerolog.Nop(), Options{
		Driver:          DriverLog,
		FromName:        "App",
		FromAddress:     "no-reply@example.test",
		ReplyTo:         "support@example.test",
		Headers:         map[string]string{"x-mailer": "app", "X-Campaign": "default"},
		MessageIDDomain: "example.test",
		Transport:       transport,
	}, bundle)
	require.NoError(t, err)
	return m, transport
}

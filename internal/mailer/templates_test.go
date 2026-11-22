package mailer_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/i18n"
	"github.com/barthjs/photopi/internal/mailer"
)

// usedKey matches a catalog key a template translates.
var usedKey = regexp.MustCompile(`t(?:html)? \.Locale "([^"]+)"`)

// discardTransport accepts every message, so a test sends emails without delivering them.
type discardTransport struct{}

// Send discards the message.
func (discardTransport) Send(context.Context, mailer.Message) error { return nil }

// TestTemplates renders every email in every locale and checks that no translation is missing,
// which would show the catalog key instead of text, and that every image the templates embed exists.
func TestTemplates(t *testing.T) {
	t.Parallel()

	bundle, err := i18n.New()
	require.NoError(t, err)
	m, err := mailer.New(zerolog.Nop(), mailer.Options{
		Driver:    mailer.DriverLog,
		Transport: discardTransport{},
	}, bundle)
	require.NoError(t, err)

	for _, name := range contentTemplates(t) {
		source, err := os.ReadFile(filepath.Join("templates", name))
		require.NoError(t, err)
		layout, err := os.ReadFile(filepath.Join("templates", "base_html.gohtml"))
		require.NoError(t, err)

		var keys []string
		for _, match := range usedKey.FindAllStringSubmatch(string(source)+string(layout), -1) {
			keys = append(keys, match[1])
		}

		for _, locale := range locales(t) {
			t.Run("renders "+name+" in "+locale, func(t *testing.T) {
				t.Parallel()

				subject, plain, html, err := m.RenderOnly(name, sampleData(locale))

				require.NoError(t, err)
				assert.NotEmpty(t, subject)
				assert.Contains(t, plain, "Sample App")
				assert.Contains(t, html, "Sample App")
				assert.Contains(t, html, `lang="`+locale+`"`)
				for _, key := range keys {
					assert.NotContains(t, subject+plain+html, key, "translation of %q", key)
				}

				require.NoError(t, m.Send(t.Context(), "to@example.test", name, sampleData(locale)), "the email and its inline images")
			})
		}
	}
}

// contentTemplates lists the email templates, without the shared layouts.
func contentTemplates(t *testing.T) []string {
	t.Helper()

	files, err := filepath.Glob(filepath.Join("templates", "*.gohtml"))
	require.NoError(t, err)

	var names []string
	for _, file := range files {
		name := filepath.Base(file)
		if !strings.HasPrefix(name, "base_") {
			names = append(names, name)
		}
	}
	require.NotEmpty(t, names)
	return names
}

// locales lists the locales of the translation catalog.
func locales(t *testing.T) []string {
	t.Helper()

	files, err := filepath.Glob(filepath.Join("..", "i18n", "locales", "*.json"))
	require.NoError(t, err)

	var names []string
	for _, file := range files {
		names = append(names, strings.TrimSuffix(filepath.Base(file), ".json"))
	}
	require.NotEmpty(t, names)
	return names
}

package mailer

import (
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/i18n"
)

const (
	testHTMLLayout  = `{{- define "baseHTML" -}}<html lang="{{ .Locale }}">{{ template "content_html" . }}</html>{{- end -}}`
	testPlainLayout = `{{- define "basePlain" -}}{{ template "content_plain" . }}{{- end -}}`
	testContent     = `{{ define "subject" }} {{ t .Locale "mail.subject" . }} {{ end }}
{{ define "content_html" }}<p>{{ t .Locale "mail.greeting" . }}</p><p>{{ thtml .Locale "mail.markup" . }}</p>{{ end }}
{{ define "content_plain" }}{{ t .Locale "mail.greeting" . }} {{ time .At }} {{ date .Locale .At }} {{ datetime .Locale .At }} {{ number .Locale 1234.5 }} {{ tn .Locale "mail.photos" 2 nil }} {{ year }}{{ end }}`
)

// renderData is the data the test templates render.
type renderData struct {
	Locale string
	Name   string
	At     time.Time
}

// newTestRenderer creates a renderer on the test templates and catalog.
func newTestRenderer(t *testing.T) *renderer {
	t.Helper()

	r, err := newRenderer(testBundle(t), testTemplates(testContent), "example.test")
	require.NoError(t, err)
	return r
}

// testBundle returns a catalog with the keys of the test templates.
func testBundle(t *testing.T) *i18n.Bundle {
	t.Helper()

	bundle, err := i18n.NewFS(fstest.MapFS{
		"locales/en.json": {
			Data: []byte(`{"mail": {"subject": "Hi {{.Name}}", "greeting": "Hello {{.Name}}", "markup": "<b>bold</b>", "photos": {"one": "{{.Count}} photo", "other": "{{.Count}} photos"}}}`),
		},
		"locales/de.json": {
			Data: []byte(`{"mail": {"subject": "Hallo {{.Name}}", "greeting": "Guten Tag {{.Name}}", "markup": "<b>fett</b>", "photos": {"one": "{{.Count}} Foto", "other": "{{.Count}} Fotos"}}}`),
		},
	})
	require.NoError(t, err)
	return bundle
}

// testTemplates returns the layouts and one content template, laid out like the embedded templates.
func testTemplates(content string) fstest.MapFS {
	return fstest.MapFS{
		"templates/base_html.gohtml": {
			Data: []byte(testHTMLLayout),
		},
		"templates/base_plain.gohtml": {
			Data: []byte(testPlainLayout),
		},
		"templates/welcome.gohtml": {
			Data: []byte(content),
		},
		"templates/notes.txt": {
			Data: []byte("not a template"),
		},
	}
}

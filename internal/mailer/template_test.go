package mailer

import (
	"strconv"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRenderer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fsys    fstest.MapFS
		wantErr string
	}{
		{
			name:    "rejects invalid syntax",
			fsys:    testTemplates(`{{ define "subject" }}`),
			wantErr: `parsing html template "welcome.gohtml"`,
		},
		{
			name: "rejects invalid html",
			fsys: func() fstest.MapFS {
				fsys := testTemplates(testContent)
				fsys["templates/base_html.gohtml"] = &fstest.MapFile{
					Data: []byte(`{{ define "baseHTML" }}<a href="{{ .URL }}`),
				}
				return fsys
			}(),
			wantErr: "parsing html template",
		},
		{
			name: "rejects an invalid plain layout",
			fsys: func() fstest.MapFS {
				fsys := testTemplates(testContent)
				fsys["templates/base_plain.gohtml"] = &fstest.MapFile{
					Data: []byte(`{{ define "basePlain" }}{{ unknownFunc }}{{ end }}`),
				}
				return fsys
			}(),
			wantErr: "parsing text template",
		},
		{
			name:    "rejects a missing templates directory",
			fsys:    fstest.MapFS{},
			wantErr: "templates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := newRenderer(testBundle(t), tt.fsys, "example.test")

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestRenderer_render(t *testing.T) {
	t.Parallel()

	t.Run("renders all parts", func(t *testing.T) {
		t.Parallel()

		r := newTestRenderer(t)
		at := time.Date(2026, 5, 1, 9, 30, 0, 0, time.UTC)

		subject, plain, html, err := r.render("welcome.gohtml", renderData{
			Locale: "de",
			Name:   "Ada <Admin>",
			At:     at,
		})

		require.NoError(t, err)
		assert.Equal(t, "Hallo Ada <Admin>", subject, "the subject is trimmed plain text")
		assert.Equal(t, "Guten Tag Ada <Admin> 09:30 01.05.2026 01.05.2026, 09:30 1.234,5 2 Fotos "+strconv.Itoa(time.Now().Year()), plain)
		assert.Equal(t, `<html lang="de"><p>Guten Tag Ada &lt;Admin&gt;</p><p><b>fett</b></p></html>`, html,
			"t is escaped in HTML, thtml is not")
	})

	t.Run("renders missing values as nothing", func(t *testing.T) {
		t.Parallel()

		content := `{{ define "subject" }}[{{ .Name }}]{{ end }}` +
			`{{ define "content_html" }}<p>[{{ .Name }}][{{ .User.Name }}]</p>{{ end }}` +
			`{{ define "content_plain" }}[{{ .Name }}][{{ .User.Name }}][{{ t .Locale "mail.greeting" . }}]{{ end }}`
		r, err := newRenderer(testBundle(t), testTemplates(content), "example.test")
		require.NoError(t, err)

		subject, plain, html, err := r.render("welcome.gohtml", map[string]any{"Locale": "en"})

		require.NoError(t, err)
		assert.Equal(t, "[]", subject)
		assert.Equal(t, "[][][Hello ]", plain, "missing values and missing placeholders of messages stay empty")
		assert.Equal(t, `<html lang="en"><p>[][]</p></html>`, html)
	})

	t.Run("renders the content id of an image", func(t *testing.T) {
		t.Parallel()

		content := `{{ define "subject" }}Subject{{ end }}` +
			`{{ define "content_html" }}<img src="{{ cid "logo.png" }}" alt="">{{ end }}` +
			`{{ define "content_plain" }}Body{{ end }}`
		r, err := newRenderer(testBundle(t), testTemplates(content), "example.test")
		require.NoError(t, err)

		_, _, html, err := r.render("welcome.gohtml", map[string]any{"Locale": "en"})

		require.NoError(t, err)
		assert.Contains(t, html, `src="cid:logo.png@example.test"`)
	})

	t.Run("rejects an unknown template", func(t *testing.T) {
		t.Parallel()

		r := newTestRenderer(t)

		_, _, _, err := r.render("missing.gohtml", nil)

		require.Error(t, err)
		assert.ErrorContains(t, err, `mail template "missing.gohtml" not found`)
	})

	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "rejects a missing subject",
			content: `{{ define "content_html" }}{{ end }}{{ define "content_plain" }}{{ end }}`,
			wantErr: "rendering subject",
		},
		{
			name:    "rejects a failing plain body",
			content: `{{ define "subject" }}s{{ end }}{{ define "content_html" }}{{ end }}{{ define "content_plain" }}{{ .Missing.Field }}{{ end }}`,
			wantErr: "rendering plain body",
		},
		{
			name:    "rejects a failing html body",
			content: `{{ define "subject" }}s{{ end }}{{ define "content_plain" }}{{ end }}{{ define "content_html" }}{{ .Missing.Field }}{{ end }}`,
			wantErr: "rendering html body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, err := newRenderer(testBundle(t), testTemplates(tt.content), "example.test")
			require.NoError(t, err)

			_, _, _, err = r.render("welcome.gohtml", renderData{
				Locale: "en",
			})

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}

	t.Run("rejects an invalid image name", func(t *testing.T) {
		t.Parallel()

		invalid := `{{ define "subject" }}S{{ end }}{{ define "content_html" }}{{ cid "../x.png" }}{{ end }}{{ define "content_plain" }}B{{ end }}`
		r, err := newRenderer(testBundle(t), testTemplates(invalid), "example.test")
		require.NoError(t, err)

		_, _, _, err = r.render("welcome.gohtml", map[string]any{"Locale": "en"})

		assert.ErrorIs(t, err, ErrInvalidImageName)
	})
}

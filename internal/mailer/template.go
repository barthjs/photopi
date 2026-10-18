package mailer

import (
	"bytes"
	"embed"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"maps"
	"path/filepath"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/barthjs/photopi/internal/i18n"
)

// templateFS holds the embedded email templates and their images.
//
//go:embed templates
var templateFS embed.FS

// Each email is rendered against a shared HTML layout and a shared plain-text layout.
const (
	htmlLayout  = "templates/base_html.gohtml"
	plainLayout = "templates/base_plain.gohtml"
)

// mailTemplate holds the parsed HTML and plain-text variants of one email.
type mailTemplate struct {
	html *htmltemplate.Template
	text *texttemplate.Template
}

// renderer renders the embedded email templates against the message catalog.
type renderer struct {
	templates map[string]mailTemplate
}

// newRenderer parses every content template of fsys against both layouts and makes missing values render as nothing.
// The mailer passes the embedded templates with those of the templates package layered over them, tests their own.
// The template function cid turns an image name into the "cid:" URL of an inline image
// with the given Content-ID domain, see InlineImage.
func newRenderer(bundle *i18n.Bundle, fsys fs.FS, contentIDDomain string) (*renderer, error) {
	// The functions that depend on the language take the locale first, as in {{ date .Locale .At }},
	// since one template serves every locale.
	funcs := map[string]any{
		"t":  bundle.T,
		"tn": bundle.TN,
		"thtml": func(locale, key string, data any) htmltemplate.HTML {
			return htmltemplate.HTML(bundle.T(locale, key, data))
		},
		"number":   i18n.FormatNumber,
		"date":     i18n.FormatDate,
		"datetime": i18n.FormatDateTime,
		"time":     func(t time.Time) string { return t.Format("15:04") },
		"year":     func() int { return time.Now().Year() },
		"cid":      cidFunc(contentIDDomain),
	}
	maps.Copy(funcs, i18n.TemplateFuncs())

	var contents []string
	err := fs.WalkDir(fsys, "templates", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Ext(path) == ".gohtml" {
			contents = append(contents, path)
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	htmlLayoutName := filepath.Base(htmlLayout)
	plainLayoutName := filepath.Base(plainLayout)

	templates := make(map[string]mailTemplate)
	for _, content := range contents {
		name := filepath.Base(content)
		if name == htmlLayoutName || name == plainLayoutName {
			continue
		}

		htmlTmpl, err := htmltemplate.New(name).Funcs(funcs).ParseFS(fsys, htmlLayout, content)
		if err != nil {
			return nil, fmt.Errorf("parsing html template %q: %w", name, err)
		}

		textTmpl, err := texttemplate.New(name).Funcs(funcs).ParseFS(fsys, plainLayout, content)
		if err != nil {
			return nil, fmt.Errorf("parsing text template %q: %w", name, err)
		}

		for _, each := range htmlTmpl.Templates() {
			i18n.BlankMissing(each.Tree)
		}
		for _, each := range textTmpl.Templates() {
			i18n.BlankMissing(each.Tree)
		}

		templates[name] = mailTemplate{
			html: htmlTmpl,
			text: textTmpl,
		}
	}

	return &renderer{
		templates: templates,
	}, nil
}

// render produces the subject line and the plain-text and HTML bodies of the named template with the given data.
func (r *renderer) render(name string, data any) (subject, plain, html string, err error) {
	tmpl, ok := r.templates[name]
	if !ok {
		return "", "", "", fmt.Errorf("mail template %q not found", name)
	}

	var subjectBuf bytes.Buffer
	if err := tmpl.text.ExecuteTemplate(&subjectBuf, "subject", data); err != nil {
		return "", "", "", fmt.Errorf("rendering subject: %w", err)
	}

	var plainBuf bytes.Buffer
	if err := tmpl.text.ExecuteTemplate(&plainBuf, "basePlain", data); err != nil {
		return "", "", "", fmt.Errorf("rendering plain body: %w", err)
	}

	var htmlBuf bytes.Buffer
	if err := tmpl.html.ExecuteTemplate(&htmlBuf, "baseHTML", data); err != nil {
		return "", "", "", fmt.Errorf("rendering html body: %w", err)
	}

	return strings.TrimSpace(subjectBuf.String()), plainBuf.String(), htmlBuf.String(), nil
}

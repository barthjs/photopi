package service

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const defaultEventSlug = "event"

var germanTransliterations = strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss", "Ä", "Ae", "Ö", "Oe", "Ü", "Ue")

// slugify converts s into a lowercase slug that is safe for file names and URLs. Umlauts are transliterated,
// other accents dropped, and every run of other characters becomes one hyphen. An empty slug is defaultEventSlug.
func slugify(s string) string {
	s = germanTransliterations.Replace(s)
	s = removeDiacritics(s)
	s = strings.ToLower(s)

	var b strings.Builder
	lastWasHyphen := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			_, _ = b.WriteRune(r)
			lastWasHyphen = false
		default:
			if !lastWasHyphen && b.Len() > 0 {
				_ = b.WriteByte('-')
				lastWasHyphen = true
			}
		}
	}

	result := strings.TrimSuffix(b.String(), "-")
	if result == "" {
		return defaultEventSlug
	}
	return result
}

// removeDiacritics drops the accents of letters with a base letter, so "Fête" becomes "Fete".
// The transform chain keeps state while it runs, so every call builds its own.
func removeDiacritics(s string) string {
	diacritics := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, err := transform.String(diacritics, s)
	if err != nil {
		return s
	}
	return result
}

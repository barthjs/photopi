package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlugify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercases words",
			input: "Summer Party",
			want:  "summer-party",
		},
		{
			name:  "transliterates german umlauts",
			input: "Größe Übung Ärger Öl",
			want:  "groesse-uebung-aerger-oel",
		},
		{
			name:  "collapses repeated separators",
			input: "a  --  b",
			want:  "a-b",
		},
		{
			name:  "trims leading and trailing separators",
			input: "  !Hello!  ",
			want:  "hello",
		},
		{
			name:  "keeps digits",
			input: "Party 2026",
			want:  "party-2026",
		},
		{
			name:  "drops accents",
			input: "Fête à Paris",
			want:  "fete-a-paris",
		},
		{
			name:  "drops accents of other languages",
			input: "Crème brûlée Øresund Łódź",
			want:  "creme-brulee-resund-odz",
		},
		{
			name:  "turns letters without a base letter into separators",
			input: "Ørsted",
			want:  "rsted",
		},
		{
			name:  "returns the default slug for an empty text",
			input: "",
			want:  defaultEventSlug,
		},
		{
			name:  "returns the default slug for only symbols",
			input: "!!!",
			want:  defaultEventSlug,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			slug := slugify(tt.input)

			assert.Equal(t, tt.want, slug)
		})
	}
}

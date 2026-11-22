package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
)

func TestValidateRGBColor(t *testing.T) {
	t.Parallel()

	type request struct {
		Color string `json:"color" validate:"rgbcolor"`
	}

	tests := []struct {
		name  string
		color string
		valid bool
	}{
		{
			name:  "accepts six hex digits",
			color: apitest.Faker(t).HexColor(),
			valid: true,
		},
		{
			name:  "accepts upper case hex digits",
			color: "#A1B2C3",
			valid: true,
		},
		{
			name:  "rejects the short form",
			color: "#fff",
			valid: false,
		},
		{
			name:  "rejects an alpha channel",
			color: "#a1b2c3ff",
			valid: false,
		},
		{
			name:  "rejects a color without hash",
			color: "a1b2c3",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			errs, err := newBareAPI().validate(t.Context(), request{
				Color: tt.color,
			})
			require.NoError(t, err)
			if tt.valid {
				assert.Nil(t, errs)
				return
			}
			assert.Equal(t, FieldErrors{
				"color": {
					{
						Tag: "rgbcolor",
					},
				},
			}, errs)
		})
	}
}

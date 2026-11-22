package mailer

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListUnsubscribe_IsZero(t *testing.T) {
	t.Parallel()

	t.Run("ignores one-click without a target", func(t *testing.T) {
		t.Parallel()

		unsubscribe := ListUnsubscribe{
			OneClick: true,
		}

		isZero := unsubscribe.IsZero()

		assert.True(t, isZero)
	})
}

func TestListUnsubscribe_header(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		unsubscribe ListUnsubscribe
		want        string
	}{
		{
			name: "returns the address",
			unsubscribe: ListUnsubscribe{
				Mailto: "u@example.test",
			},
			want: "<mailto:u@example.test>",
		},
		{
			name: "returns the address and the URL",
			unsubscribe: ListUnsubscribe{
				Mailto: "u@example.test",
				URL:    "https://example.test/u",
			},
			want: "<mailto:u@example.test>, <https://example.test/u>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			header := tt.unsubscribe.header()

			assert.Equal(t, tt.want, header)
		})
	}
}

func TestNewTransport(t *testing.T) {
	t.Parallel()

	override := &recordingTransport{}
	tests := []struct {
		name    string
		options Options
		want    any
		wantErr string
	}{
		{
			name: "creates the log transport",
			options: Options{
				Driver: DriverLog,
			},
			want: &logTransport{},
		},
		{
			name: "creates the smtp transport",
			options: Options{
				Driver:   DriverSMTP,
				SMTPHost: "127.0.0.1",
				SMTPPort: 25,
			},
			want: &smtpTransport{},
		},
		{
			name: "prefers the override over the driver",
			options: Options{
				Driver:    DriverSMTP,
				Transport: override,
			},
			want: override,
		},
		{
			name: "rejects an unknown driver",
			options: Options{
				Driver: "pigeon",
			},
			wantErr: `mailer: unknown driver "pigeon"`,
		},
		{
			name: "rejects smtp without a host",
			options: Options{
				Driver: DriverSMTP,
			},
			wantErr: "creating mail client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			transport, err := newTransport(zerolog.Nop(), tt.options)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.want == override {
				assert.Same(t, override, transport)
				return
			}
			assert.IsType(t, tt.want, transport)
		})
	}
}

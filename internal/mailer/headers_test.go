package mailer

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeHeaders(t *testing.T) {
	t.Parallel()

	t.Run("prefers the per-email headers regardless of case", func(t *testing.T) {
		t.Parallel()

		merged, err := mergeHeaders(map[string]string{"X-Mailer": "app", "x-campaign": "global"}, map[string]string{"X-CAMPAIGN": "email"})

		require.NoError(t, err)
		assert.Equal(t, map[string]string{"X-Mailer": "app", "X-Campaign": "email"}, merged)
	})

	t.Run("returns nil without headers", func(t *testing.T) {
		t.Parallel()

		merged, err := mergeHeaders(nil, map[string]string{})

		require.NoError(t, err)
		assert.Nil(t, merged)
	})

	t.Run("rejects an invalid global header", func(t *testing.T) {
		t.Parallel()

		_, err := mergeHeaders(map[string]string{"From": "a@example.test"}, nil)

		require.ErrorIs(t, err, ErrInvalidHeader)
	})

	t.Run("rejects an invalid per-email header", func(t *testing.T) {
		t.Parallel()

		_, err := mergeHeaders(nil, map[string]string{"X-Bad": "a\nb"})

		require.ErrorIs(t, err, ErrInvalidHeader)
	})
}

func TestCheckHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		header    string
		value     string
		canonical string
		wantErr   bool
	}{
		{
			name:      "returns the canonical name",
			header:    "x-campaign-id",
			value:     "spring",
			canonical: "X-Campaign-Id",
		},
		{
			name:      "accepts an empty value",
			header:    "X-Empty",
			value:     "",
			canonical: "X-Empty",
		},
		{
			name:    "rejects an empty name",
			header:  "",
			value:   "v",
			wantErr: true,
		},
		{
			name:    "rejects a space in the name",
			header:  "X Bad",
			value:   "v",
			wantErr: true,
		},
		{
			name:    "rejects a colon in the name",
			header:  "X:Bad",
			value:   "v",
			wantErr: true,
		},
		{
			name:    "rejects a non-ASCII name",
			header:  "X-Café",
			value:   "v",
			wantErr: true,
		},
		{
			name:    "rejects a line break in the value",
			header:  "X-Inject",
			value:   "a\r\nBcc: evil@example.test",
			wantErr: true,
		},
		{
			name:    "rejects a header with a dedicated field",
			header:  "reply-to",
			value:   "a@example.test",
			wantErr: true,
		},
		{
			name:    "rejects the unsubscribe header",
			header:  "List-Unsubscribe",
			value:   "<mailto:a@example.test>",
			wantErr: true,
		},
		{
			name:    "rejects the message id",
			header:  "Message-ID",
			value:   "x@example.test",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			canonical, err := checkHeader(tt.header, tt.value)

			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidHeader)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.canonical, canonical)
		})
	}
}

func TestCheckListUnsubscribe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   ListUnsubscribe
		wantErr bool
	}{
		{
			name:  "accepts no unsubscribe method",
			value: ListUnsubscribe{},
		},
		{
			name: "accepts an address",
			value: ListUnsubscribe{
				Mailto: "unsubscribe@example.test",
			},
		},
		{
			name: "accepts one-click with a URL",
			value: ListUnsubscribe{
				URL:      "https://example.test/u/1",
				OneClick: true,
			},
		},
		{
			name: "rejects one-click without a URL",
			value: ListUnsubscribe{
				Mailto:   "unsubscribe@example.test",
				OneClick: true,
			},
			wantErr: true,
		},
		{
			name: "rejects a plain http URL",
			value: ListUnsubscribe{
				URL: "http://example.test/u",
			},
			wantErr: true,
		},
		{
			name: "rejects a URL without host",
			value: ListUnsubscribe{
				URL: "https:///u",
			},
			wantErr: true,
		},
		{
			name: "rejects an unparsable URL",
			value: ListUnsubscribe{
				URL: "https://exa mple.test/%zz",
			},
			wantErr: true,
		},
		{
			name: "rejects a URL with angle brackets",
			value: ListUnsubscribe{
				URL: "https://example.test/u>",
			},
			wantErr: true,
		},
		{
			name: "rejects an address with a line break",
			value: ListUnsubscribe{
				Mailto: "a@example.test\r\nBcc: b@example.test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := checkListUnsubscribe(tt.value)

			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidHeader)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestCheckDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		domain  string
		wantErr bool
	}{
		{
			name:   "accepts a host name",
			domain: "mail.example.test",
		},
		{
			name:   "accepts an empty domain",
			domain: "",
		},
		{
			name:    "rejects an address",
			domain:  "a@example.test",
			wantErr: true,
		},
		{
			name:    "rejects a line break",
			domain:  "example.test\r\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := checkDomain(tt.domain)

			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidHeader)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestNewMessageID(t *testing.T) {
	t.Parallel()

	t.Run("returns a unique id at the domain", func(t *testing.T) {
		t.Parallel()

		first, err := newMessageID("example.test")
		require.NoError(t, err)
		second, err := newMessageID("example.test")
		require.NoError(t, err)

		assert.Regexp(t, regexp.MustCompile(`^[0-9a-f]{32}@example\.test$`), first)
		assert.NotEqual(t, first, second)
	})
}

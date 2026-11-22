package mailer

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContentIDDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		messageIDDomain string
		from            string
		want            string
	}{
		{
			name:            "returns the message id domain",
			messageIDDomain: "mail.example.test",
			from:            "a@example.test",
			want:            "mail.example.test",
		},
		{
			name: "returns the sender domain",
			from: "a@example.test",
			want: "example.test",
		},
		{
			name: "returns the default for a sender without domain",
			from: "a@",
			want: defaultContentIDDomain,
		},
		{
			name: "returns the default without settings",
			want: defaultContentIDDomain,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			domain := contentIDDomain(tt.messageIDDomain, tt.from)

			assert.Equal(t, tt.want, domain)
		})
	}
}

func TestCidFunc(t *testing.T) {
	t.Parallel()

	t.Run("returns the content id URL", func(t *testing.T) {
		t.Parallel()

		cid := cidFunc("example.test")

		url, err := cid("logo.png")

		require.NoError(t, err)
		assert.Equal(t, "cid:logo.png@example.test", string(url))
	})

	tests := []struct {
		name  string
		image string
	}{
		{
			name:  "rejects an empty name",
			image: "",
		},
		{
			name:  "rejects a parent path",
			image: "../logo.png",
		},
		{
			name:  "rejects a subdirectory",
			image: "images/logo.png",
		},
		{
			name:  "rejects a hidden name",
			image: ".hidden",
		},
		{
			name:  "rejects a space in the name",
			image: "logo png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cid := cidFunc("example.test")

			_, err := cid(tt.image)

			assert.ErrorIs(t, err, ErrInvalidImageName, "name %q", tt.image)
		})
	}
}

func TestImageResolver_resolve(t *testing.T) {
	t.Parallel()

	logo := testPNG(t)
	images := fstest.MapFS{
		"templates/images/logo.png": {
			Data: logo,
		},
		"templates/images/icon.png": {
			Data: logo,
		},
		"templates/images/note.txt": {
			Data: []byte("plain text"),
		},
	}
	resolver := newImageResolver("example.test", images)

	t.Run("attaches only the referenced images once in order of use", func(t *testing.T) {
		t.Parallel()

		html := `<img src="cid:logo.png@example.test"><img src="cid:photo.jpg@example.test"><img src="cid:logo.png@example.test">`
		photo := InlineImage{
			Name:        "photo.jpg",
			Data:        logo,
			ContentType: "image/jpeg",
		}

		embedded, err := resolver.resolve(html, []InlineImage{
			photo,
			{
				Name: "unused.png",
				Data: logo,
			},
		})

		require.NoError(t, err)
		assert.Equal(t, []EmbeddedImage{
			{
				Name:        "logo.png",
				ContentID:   "logo.png@example.test",
				ContentType: "image/png",
				Data:        logo,
			},
			{
				Name:        "photo.jpg",
				ContentID:   "photo.jpg@example.test",
				ContentType: "image/jpeg",
				Data:        logo,
			},
		}, embedded)
	})

	t.Run("prefers an image of the email over a template image", func(t *testing.T) {
		t.Parallel()

		own := InlineImage{
			Name: "logo.png",
			Data: []byte("GIF89a own logo"),
		}

		embedded, err := resolver.resolve(`<img src="cid:logo.png@example.test">`, []InlineImage{own})

		require.NoError(t, err)
		require.Len(t, embedded, 1)
		assert.Equal(t, own.Data, embedded[0].Data)
		assert.Equal(t, "image/gif", embedded[0].ContentType)
	})

	t.Run("ignores references of another domain", func(t *testing.T) {
		t.Parallel()

		embedded, err := resolver.resolve(`<img src="cid:logo.png@other.test">`, nil)

		require.NoError(t, err)
		assert.Empty(t, embedded)
	})

	tests := []struct {
		name string
		html string
		own  []InlineImage
		err  error
	}{
		{
			name: "rejects an unknown image",
			html: `cid:missing.png@example.test`,
			err:  ErrImageNotFound,
		},
		{
			name: "rejects a template file that is no image",
			html: `cid:note.txt@example.test`,
			err:  ErrNotImage,
		},
		{
			name: "rejects own data that is no image",
			html: `cid:x.png@example.test`,
			own: []InlineImage{
				{
					Name: "x.png",
					Data: []byte("text"),
				},
			},
			err: ErrNotImage,
		},
		{
			name: "rejects an own content type that is no image",
			html: `cid:x.png@example.test`,
			own: []InlineImage{
				{
					Name:        "x.png",
					Data:        logo,
					ContentType: "application/pdf",
				},
			},
			err: ErrNotImage,
		},
		{
			name: "rejects a malformed content type",
			html: `cid:x.png@example.test`,
			own: []InlineImage{
				{
					Name:        "x.png",
					Data:        logo,
					ContentType: "image/",
				},
			},
			err: ErrNotImage,
		},
		{
			name: "rejects an own image with an invalid name",
			own: []InlineImage{
				{
					Name: "../x.png",
					Data: logo,
				},
			},
			err: ErrInvalidImageName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := resolver.resolve(tt.html, tt.own)

			assert.ErrorIs(t, err, tt.err)
		})
	}
}

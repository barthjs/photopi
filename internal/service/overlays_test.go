package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/store"
)

func TestOverlayService_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the overlays", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		f := apitest.Faker(t)
		svc := newTestOverlayService(t, OverlayOptions{})
		second, err := svc.Create(ctx, "zeta frame.png", []byte(f.LoremIpsumSentence(6)))
		require.NoError(t, err)
		first, err := svc.Create(ctx, "alpha.png", []byte(f.LoremIpsumSentence(6)))
		require.NoError(t, err)

		overlays, err := svc.List(ctx)

		require.NoError(t, err)
		assert.Equal(t, []domain.Overlay{
			first,
			second,
		}, overlays)
	})
}

func TestOverlayService_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the overlay", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc := newTestOverlayService(t, OverlayOptions{})
		content := []byte(apitest.Faker(t).LoremIpsumSentence(6))
		overlay, err := svc.Create(ctx, "alpha.png", content)
		require.NoError(t, err)

		data, err := svc.Get(ctx, overlay.Path)

		require.NoError(t, err)
		assert.Equal(t, content, data)
	})
}

func TestOverlayService_Create(t *testing.T) {
	t.Parallel()

	t.Run("stores the overlay", func(t *testing.T) {
		t.Parallel()

		svc := newTestOverlayService(t, OverlayOptions{})

		overlay, err := svc.Create(t.Context(), "zeta frame.png", []byte(apitest.Faker(t).LoremIpsumSentence(6)))

		require.NoError(t, err)
		assert.Equal(t, domain.Overlay{
			Name: "zeta-frame.png",
			Path: "overlays/zeta-frame.png",
		}, overlay)
	})

	t.Run("keeps the overlay inside its folder", func(t *testing.T) {
		t.Parallel()

		svc := newTestOverlayService(t, OverlayOptions{})

		overlay, err := svc.Create(t.Context(), "../alpha.png", []byte(apitest.Faker(t).LoremIpsumSentence(6)))

		require.NoError(t, err)
		assert.Equal(t, "overlays/alpha.png", overlay.Path, "the name cannot leave the overlay folder")
	})

	t.Run("stores the overlay with the png extension", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			desc     string
			filename string
			want     string
		}{
			{
				desc:     "adds a missing extension",
				filename: "frame",
				want:     "frame.png",
			},
			{
				desc:     "lowers an uppercase extension",
				filename: "frame.PNG",
				want:     "frame.png",
			},
			{
				desc:     "adds the extension to another one",
				filename: "frame.jpg",
				want:     "frame.jpg.png",
			},
		}

		for _, tt := range tests {
			t.Run(tt.desc, func(t *testing.T) {
				t.Parallel()
				svc := newTestOverlayService(t, OverlayOptions{})

				overlay, err := svc.Create(t.Context(), tt.filename, []byte(apitest.Faker(t).LoremIpsumSentence(6)))

				require.NoError(t, err)
				assert.Equal(t, domain.Overlay{
					Name: tt.want,
					Path: "overlays/" + tt.want,
				}, overlay)
			})
		}
	})

	for _, name := range []string{"   ", "..", ".png", "..png"} {
		t.Run("names an overlay without a usable name "+name, func(t *testing.T) {
			t.Parallel()

			svc := newTestOverlayService(t, OverlayOptions{})

			overlay, err := svc.Create(t.Context(), name, []byte("x"))

			require.NoError(t, err)
			assert.True(t, strings.HasPrefix(overlay.Path, "overlays/overlay-"), overlay.Path)
			assert.True(t, strings.HasSuffix(overlay.Path, ".png"), overlay.Path)
		})
	}

	t.Run("rejects a name that is too long", func(t *testing.T) {
		t.Parallel()

		svc := newTestOverlayService(t, OverlayOptions{})

		_, err := svc.Create(t.Context(), strings.Repeat("a", 300)+".png", []byte("x"))

		assertValidation(t, err, "file", "invalid")
	})
}

func TestOverlayService_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes the overlay", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc := newTestOverlayService(t, OverlayOptions{})
		overlay, err := svc.Create(ctx, "alpha.png", []byte(apitest.Faker(t).LoremIpsumSentence(6)))
		require.NoError(t, err)

		err = svc.Delete(ctx, overlay.Name)

		require.NoError(t, err)
		_, err = svc.Get(ctx, overlay.Path)
		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("returns not found for an unknown overlay", func(t *testing.T) {
		t.Parallel()

		svc := newTestOverlayService(t, OverlayOptions{})

		err := svc.Delete(t.Context(), apitest.Faker(t).LetterN(8)+".png")

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("returns not found for a malformed name", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc := newTestOverlayService(t, OverlayOptions{})
		overlay, err := svc.Create(ctx, "alpha.png", []byte(apitest.Faker(t).LoremIpsumSentence(6)))
		require.NoError(t, err)

		for _, name := range append(invalidOverlayNames(), "../alpha.png", "x/../alpha.png") {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				err := svc.Delete(t.Context(), name)

				assert.ErrorIs(t, err, apperr.ErrNotFound)
				overlays, err := svc.List(t.Context())
				require.NoError(t, err)
				assert.Equal(t, []domain.Overlay{
					overlay,
				}, overlays, "a malformed name deletes nothing")
			})
		}
	})

	t.Run("keeps the active overlay", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "overlays/frame.png", []byte("x"))
		svc := NewOverlayService(OverlayOptions{
			ActivePath: "overlays/frame.png",
		}, store.NewOverlayStore(storage))

		err := svc.Delete(t.Context(), "frame.png")

		assert.ErrorIs(t, err, ErrOverlayActive)
		assert.Equal(t, []string{"overlays/frame.png"}, listFiles(t, storage, "overlays/"))
	})
}

func TestOverlayService_Download(t *testing.T) {
	t.Parallel()

	t.Run("returns the overlay", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		content := []byte(apitest.Faker(t).LoremIpsumSentence(6))
		svc := newTestOverlayService(t, OverlayOptions{})
		overlay, err := svc.Create(ctx, "alpha.png", content)
		require.NoError(t, err)

		data, err := svc.Download(ctx, overlay.Name)

		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("returns not found for an invalid or unknown name", func(t *testing.T) {
		t.Parallel()

		svc := newTestOverlayService(t, OverlayOptions{})

		for _, name := range []string{"missing.png", "../alpha.png", "alpha.jpg"} {
			_, err := svc.Download(t.Context(), name)

			assert.ErrorIs(t, err, apperr.ErrNotFound, name)
		}
	})
}

// invalidOverlayNames returns names that cannot address an overlay file.
func invalidOverlayNames() []string {
	return []string{"", "alpha", " .png", "..png", "../escape.png", "a/b.png", `a\b.png`, strings.Repeat("a", 252) + ".png"}
}

// newTestOverlayService builds an overlay service on a fresh local storage.
func newTestOverlayService(t *testing.T, cfg OverlayOptions) *OverlayService {
	t.Helper()

	return NewOverlayService(cfg, store.NewOverlayStore(newTestStorage(t)))
}

func TestSanitizeFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "keeps a plain name",
			input: "frame.png",
			want:  "frame.png",
		},
		{
			name:  "drops the path of a traversal",
			input: "../../etc/passwd",
			want:  "passwd",
		},
		{
			name:  "replaces spaces and symbols",
			input: "  my frame (1).png ",
			want:  "my-frame-1-.png",
		},
		{
			name:  "replaces non-ASCII letters",
			input: "Café-Résumé.png",
			want:  "Caf--R-sum-.png",
		},
		{
			name:  "returns nothing for blanks",
			input: "   ",
			want:  "",
		},
		{
			name:  "returns nothing for the parent directory",
			input: "..",
			want:  "",
		},
		{
			name:  "returns nothing for only slashes",
			input: "///",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			name := sanitizeFilename(tt.input)

			assert.Equal(t, tt.want, name)
		})
	}
}

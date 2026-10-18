package service

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/i18n"
	"github.com/barthjs/photopi/internal/store"
)

func TestLayoutService_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the layouts sorted by id", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc := newTestLayoutService(t)
		require.NoError(t, svc.Save(ctx, "b", testLayout("b")))
		require.NoError(t, svc.Save(ctx, "a", testLayout("a")))

		layouts, err := svc.List(ctx)

		require.NoError(t, err)
		assert.Equal(t, []domain.Layout{
			testLayout("a"),
			testLayout("b"),
		}, layouts, "sorted by ID")
	})
}

func TestLayoutService_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the layout", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc := newTestLayoutService(t)
		require.NoError(t, svc.Save(ctx, "b", testLayout("b")))

		got, err := svc.Get(ctx, "b")

		require.NoError(t, err)
		assert.Equal(t, new(testLayout("b")), got)
	})

	t.Run("returns not found for an unknown layout", func(t *testing.T) {
		t.Parallel()

		_, err := newTestLayoutService(t).Get(t.Context(), apitest.Faker(t).LetterN(8))

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("returns not found for a malformed id", func(t *testing.T) {
		t.Parallel()

		svc := newTestLayoutService(t)

		for _, id := range invalidLayoutIDs() {
			t.Run(id, func(t *testing.T) {
				t.Parallel()

				_, err := svc.Get(t.Context(), id)

				assert.ErrorIs(t, err, apperr.ErrNotFound)
			})
		}
	})
}

func TestLayoutService_Save(t *testing.T) {
	t.Parallel()

	t.Run("stores the layout", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc := newTestLayoutService(t)

		err := svc.Save(ctx, "b", testLayout("b"))

		require.NoError(t, err)
		got, err := svc.Get(ctx, "b")
		require.NoError(t, err)
		assert.Equal(t, new(testLayout("b")), got)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		svc := newTestLayoutService(t)

		for _, id := range invalidLayoutIDs() {
			t.Run(id, func(t *testing.T) {
				t.Parallel()

				err := svc.Save(t.Context(), id, testLayout(id))

				assertValidation(t, err, "id", "invalid")
				layouts, err := svc.List(t.Context())
				require.NoError(t, err)
				assert.Empty(t, layouts)
			})
		}
	})

	t.Run("rejects a layout of another id", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc := newTestLayoutService(t)

		err := svc.Save(ctx, "a", testLayout("b"))

		assertValidation(t, err, "id", "invalid")
		layouts, err := svc.List(ctx)
		require.NoError(t, err)
		assert.Empty(t, layouts, "a layout of another id is not stored")
	})
}

func TestLayoutService_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes the layout", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc := newTestLayoutService(t)
		require.NoError(t, svc.Save(ctx, "b", testLayout("b")))

		err := svc.Delete(ctx, "b")

		require.NoError(t, err)
		_, err = svc.Get(ctx, "b")
		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("returns not found for an unknown layout", func(t *testing.T) {
		t.Parallel()

		svc := newTestLayoutService(t)

		err := svc.Delete(t.Context(), apitest.Faker(t).LetterN(8))

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("returns not found for a malformed id", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc := newTestLayoutService(t)
		require.NoError(t, svc.Save(ctx, "b", testLayout("b")))

		for _, id := range append(invalidLayoutIDs(), "x/../b") {
			t.Run(id, func(t *testing.T) {
				t.Parallel()

				err := svc.Delete(t.Context(), id)

				assert.ErrorIs(t, err, apperr.ErrNotFound)
				layouts, err := svc.List(t.Context())
				require.NoError(t, err)
				assert.Equal(t, []domain.Layout{
					testLayout("b"),
				}, layouts, "a malformed id deletes nothing")
			})
		}
	})
}

func TestLayoutService_EnsureDefaults(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	bundle, err := i18n.NewFS(fstest.MapFS{
		"locales/en.json": {
			Data: []byte(`{"layout": {"strip-3": "Strip of three"}}`),
		},
	})
	require.NoError(t, err)

	t.Run("seeds the built-in layouts with translated names", func(t *testing.T) {
		t.Parallel()

		svc := newTestLayoutService(t)

		err := svc.EnsureDefaults(ctx, bundle, "en")

		require.NoError(t, err)
		layouts, err := svc.List(ctx)
		require.NoError(t, err)
		names := map[string]string{}
		for _, layout := range layouts {
			names[layout.ID] = layout.Name
		}
		assert.Equal(t, "Strip of three", names["strip-3"])
		assert.Equal(t, "Single image (10x15 cm)", names["single-4x6"], "without a translation, the embedded name stays")
	})

	t.Run("leaves existing layouts alone", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, "layouts/own.json", marshalLayout(t, testLayout("own")))
		svc := NewLayoutService(store.NewLayoutStore(storage))

		err := svc.EnsureDefaults(ctx, bundle, "en")

		require.NoError(t, err)
		assert.Equal(t, []string{"layouts/own.json"}, listFiles(t, storage, "layouts/"))
	})
}

// newTestLayoutService builds a layout service on a fresh local storage.
func newTestLayoutService(t *testing.T) *LayoutService {
	t.Helper()

	return NewLayoutService(store.NewLayoutStore(newTestStorage(t)))
}

// invalidLayoutIDs returns IDs that cannot address a layout file.
func invalidLayoutIDs() []string {
	return []string{"", " ", ".", "..", "../escape", "a/b", `a\b`, strings.Repeat("a", 251)}
}

// testLayout returns a valid layout with one locked full-page slot.
func testLayout(id string) domain.Layout {
	return domain.Layout{
		ID:   id,
		Name: "Test " + id,
		Paper: domain.LayoutPaper{
			WidthMM:  100,
			HeightMM: 150,
		},
		Pages: []domain.LayoutPage{
			{
				Slots: []domain.LayoutSlot{
					{
						X:      0,
						Y:      0,
						W:      1,
						H:      1,
						Locked: true,
					},
				},
			},
		},
	}
}

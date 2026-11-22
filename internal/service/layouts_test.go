package service

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/i18n"
	"github.com/barthjs/photopi/internal/store"
)

func TestLayoutService_Save(t *testing.T) {
	t.Parallel()

	t.Run("rejects a layout of another name", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc := newTestLayoutService(t)

		err := svc.Save(ctx, "a", testLayout("b"))

		assertValidation(t, err, "name", "invalid")
		layouts, err := svc.List(ctx)
		require.NoError(t, err)
		assert.Empty(t, layouts, "a layout of another name is not stored")
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
		names := make([]string, 0, len(layouts))
		for _, layout := range layouts {
			names = append(names, layout.Name)
		}
		assert.Equal(t, []string{"Single image (10x15 cm)", "Strip of three"}, names, "without a translation, the embedded name stays")
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

// testLayout returns a valid layout with one locked full-page slot.
func testLayout(name string) domain.Layout {
	return domain.Layout{
		Name: name,
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

package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
	"github.com/barthjs/photopi/internal/domain"
)

func TestCollageStore_Create(t *testing.T) {
	t.Parallel()

	t.Run("stores the collage page", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		st := NewCollageStore(storage)
		collage := testCollage(3, 0)
		content := []byte(apitest.Faker(t).LoremIpsumSentence(6))

		err := st.Create(t.Context(), collage, content)

		require.NoError(t, err)
		assert.Equal(t, content, readFile(t, storage, st.Path(collage)))
	})
}

func TestCollageStore_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes the collage pages of the run", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		storage := newTestStorage(t)
		st := NewCollageStore(storage)
		require.NoError(t, st.Create(ctx, testCollage(3, 0), []byte("x")))
		require.NoError(t, st.Create(ctx, testCollage(3, 1), []byte("x")))
		require.NoError(t, st.Create(ctx, testCollage(4, 0), []byte("x")))
		putFile(t, storage, "captures/party/x_000003-01_20260101-000000.jpg", []byte("x"))

		err := st.Delete(ctx, 3)

		require.NoError(t, err)
		assert.Equal(t, []string{st.Path(testCollage(4, 0))}, listFiles(t, storage, "prints/"))
		assert.Len(t, listFiles(t, storage, "captures/"), 1, "the captures of the run stay")
	})
}

func TestCollageStore_Exists(t *testing.T) {
	t.Parallel()

	t.Run("reports stored pages", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewCollageStore(newTestStorage(t))
		collages := []domain.Collage{
			testCollage(3, 0),
			testCollage(3, 1),
		}
		for _, collage := range collages {
			require.NoError(t, st.Create(ctx, collage, []byte("x")))
		}

		exists, err := st.Exists(ctx, collages)

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("reports a missing page", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewCollageStore(newTestStorage(t))
		require.NoError(t, st.Create(ctx, testCollage(3, 0), []byte("x")))

		exists, err := st.Exists(ctx, []domain.Collage{
			testCollage(3, 0),
			testCollage(3, 1),
		})

		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestCollageStore_Path(t *testing.T) {
	t.Parallel()

	t.Run("names the page like the captures of the run", func(t *testing.T) {
		t.Parallel()

		p := NewCollageStore(newTestStorage(t)).Path(domain.Collage{
			Event:      "party",
			Prefix:     "booth_",
			Run:        3,
			Page:       1,
			LayoutHash: "0a1b2c3d",
		})

		assert.Equal(t, "prints/party/booth_000003-collage-2_0a1b2c3d.jpg", p)
	})
}

// testCollage returns a collage page of run in the event "party".
func testCollage(run, page int) domain.Collage {
	return domain.Collage{
		Event:      "party",
		Prefix:     "x",
		Run:        run,
		Page:       page,
		LayoutHash: "0a1b2c3d",
	}
}

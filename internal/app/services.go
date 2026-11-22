package app

import (
	"github.com/barthjs/photopi/internal/media"
	"github.com/barthjs/photopi/internal/store"
)

// Stores holds the stores of the application, all reading from the same file storage.
type Stores struct {
	Capture *store.CaptureStore
	Collage *store.CollageStore
	Layout  *store.LayoutStore
	Overlay *store.OverlayStore
	Run     *store.RunStore
}

// NewStores builds the stores on storage.
func NewStores(storage media.Storage) Stores {
	runFiles := store.NewRunFileIndex(storage)
	return Stores{
		Capture: store.NewCaptureStore(storage, runFiles),
		Collage: store.NewCollageStore(storage),
		Layout:  store.NewLayoutStore(storage),
		Overlay: store.NewOverlayStore(storage),
		Run:     store.NewRunStore(storage, runFiles),
	}
}

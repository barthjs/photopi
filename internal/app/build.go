package app

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"

	"github.com/barthjs/photopi/internal/cmdline"
	"github.com/barthjs/photopi/internal/config"
	"github.com/barthjs/photopi/internal/media"
	"github.com/barthjs/photopi/internal/store"
	"github.com/barthjs/photopi/internal/update"
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
	return Stores{
		Capture: store.NewCaptureStore(storage),
		Collage: store.NewCollageStore(storage),
		Layout:  store.NewLayoutStore(storage),
		Overlay: store.NewOverlayStore(storage),
		Run:     store.NewRunStore(storage),
	}
}

// NewStorage builds the configured file storage.
func NewStorage(logger zerolog.Logger, cfg config.Config) (media.Storage, error) {
	options, err := StorageOptions(cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("resolving the storage path: %w", err)
	}

	storage, err := media.New(options)
	if err != nil {
		return nil, fmt.Errorf("initializing media storage: %w", err)
	}
	if options.Driver == media.DriverLocal {
		logger.Info().Str("path", options.LocalPath).Msg("using local storage")
	}
	return storage, nil
}

// NewUpdater returns the updater for the releases of the application,
// picking the archives of the architecture the binary was built for.
func NewUpdater(version string) *update.Updater {
	return update.New(update.Options{
		Repository: config.AppRepository,
		Binary:     config.AppName,
		Version:    version,
		Arch:       cmdline.ResolveArch(),
		Token:      os.Getenv("GITHUB_TOKEN"),
	})
}

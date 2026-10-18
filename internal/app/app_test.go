package app_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/app"
	"github.com/barthjs/photopi/internal/config"
	"github.com/barthjs/photopi/internal/media"
)

// stillCamera never touches a device.
type stillCamera struct{}

// StreamPreview writes nothing.
func (stillCamera) StreamPreview(context.Context, io.Writer, string) error { return nil }

// Capture returns the smallest byte sequence framed like a JPEG.
func (stillCamera) Capture(context.Context) (*[]byte, error) {
	data := []byte{0xFF, 0xD8, 0xFF, 0xD9}
	return &data, nil
}

// noCommands refuses to run programs, so the test can never start a real one.
type noCommands struct{}

// Start refuses to run the program.
func (noCommands) Start(string, ...string) error { return os.ErrPermission }

// Output refuses to run the program.
func (noCommands) Output(context.Context, []string, string, ...string) ([]byte, error) {
	return nil, os.ErrPermission
}

// CombinedOutput refuses to run the program.
func (noCommands) CombinedOutput(context.Context, []string, string, ...string) ([]byte, error) {
	return nil, os.ErrPermission
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("builds the application", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)

		application, err := app.New(t.Context(), zerolog.Nop(), cfg, testDeps(t))

		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, application.Close()) })
		assert.Equal(t, http.StatusOK, serve(t, application.Handler(), "/api/ping").Code)
		assert.Equal(t, http.StatusOK, serve(t, application.Handler(), "/healthz").Code)
		assert.Equal(t, http.StatusOK, serve(t, application.Handler(), "/readyz").Code, "the camera is optional")
		layouts, err := os.ReadDir(filepath.Join(cfg.Storage.Path, "layouts"))
		require.NoError(t, err)
		assert.NotEmpty(t, layouts, "the default print layouts are seeded")
	})

	t.Run("uses the given storage", func(t *testing.T) {
		t.Parallel()

		storage, err := media.New(media.Options{
			Driver:    media.DriverLocal,
			LocalPath: t.TempDir(),
		})
		require.NoError(t, err)
		cfg := testConfig(t)
		cfg.Storage = unusableStorage()
		deps := testDeps(t)
		deps.Storage = storage

		application, err := app.New(t.Context(), zerolog.Nop(), cfg, deps)

		require.NoError(t, err, "the storage of the configuration is not built")
		t.Cleanup(func() { _ = application.Close() })
		paths, err := storage.List(t.Context(), "layouts/")
		require.NoError(t, err)
		assert.NotEmpty(t, paths, "the layouts end up in the given storage")
	})

	t.Run("rejects an unusable storage", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		cfg.Storage = unusableStorage()

		_, err := app.New(t.Context(), zerolog.Nop(), cfg, testDeps(t))

		require.Error(t, err)
	})

	t.Run("rejects a missing stop function", func(t *testing.T) {
		t.Parallel()

		deps := testDeps(t)
		deps.Stop = nil

		_, err := app.New(t.Context(), zerolog.Nop(), testConfig(t), deps)

		assert.ErrorContains(t, err, "Deps.Stop is required")
	})
}

// serve sends a GET request for path to handler and returns the recorded response.
func serve(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	return recorder
}

// unusableStorage returns storage settings that New cannot build a storage from.
func unusableStorage() config.StorageConfig {
	return config.StorageConfig{
		Driver: media.DriverS3,
		S3: config.S3Config{
			Endpoint:  "not a url",
			Bucket:    "b",
			AccessKey: "a",
			SecretKey: "s",
		},
	}
}

func TestApp_Close(t *testing.T) {
	t.Parallel()

	t.Run("closes the application twice", func(t *testing.T) {
		t.Parallel()

		application, err := app.New(t.Context(), zerolog.Nop(), testConfig(t), testDeps(t))
		require.NoError(t, err)
		require.NoError(t, application.Close())

		err = application.Close()

		require.NoError(t, err)
	})

	t.Run("waits for the background work", func(t *testing.T) {
		t.Parallel()

		application, err := app.New(t.Context(), zerolog.Nop(), testConfig(t), testDeps(t))
		require.NoError(t, err)
		ctx, cancel := context.WithCancel(t.Context())
		t.Cleanup(cancel)
		application.Start(ctx)

		closed := make(chan error, 1)
		go func() { closed <- application.Close() }()

		select {
		case err := <-closed:
			require.FailNow(t, "Close returned while the background work was still running", err)
		case <-time.After(200 * time.Millisecond):
		}
		cancel()
		select {
		case err := <-closed:
			assert.NoError(t, err)
		case <-time.After(5 * time.Second):
			require.FailNow(t, "Close did not return after the background work was canceled")
		}
	})
}

// testConfig returns the normalized default configuration with its storage in a temporary directory.
func testConfig(t *testing.T) config.Config {
	t.Helper()

	cfg, err := config.Defaults()
	require.NoError(t, err)
	cfg.Storage.Driver = media.DriverLocal
	cfg.Storage.Path = filepath.Join(t.TempDir(), "data")
	cfg.System = config.SystemConfig{}
	require.NoError(t, config.Validate(&cfg))
	return cfg
}

// testDeps returns dependencies that replace the camera and external programs.
func testDeps(t *testing.T) app.Deps {
	t.Helper()

	return app.Deps{
		Stop:           func() {},
		Version:        "test",
		ConfigFilePath: filepath.Join(t.TempDir(), "config.yaml"),
		Camera:         stillCamera{},
		Commands:       noCommands{},
	}
}

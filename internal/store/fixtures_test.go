package store

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/media"
)

// newTestStorage returns a local storage in a temporary directory of the test.
func newTestStorage(t *testing.T) media.Storage {
	t.Helper()

	storage, err := media.New(media.Options{
		Driver:        media.DriverLocal,
		LocalPath:     t.TempDir(),
		SignedURLBase: "/media",
	})
	require.NoError(t, err)
	return storage
}

// putFile stores data at path in storage.
func putFile(t *testing.T, storage media.Storage, storagePath string, data []byte) {
	t.Helper()

	require.NoError(t, storage.Put(t.Context(), storagePath, bytes.NewReader(data), int64(len(data))))
}

// readFile reads path from storage.
func readFile(t *testing.T, storage media.Storage, storagePath string) []byte {
	t.Helper()

	data, err := media.ReadAll(t.Context(), storage, storagePath)
	require.NoError(t, err)
	return data
}

// listFiles lists every file below prefix in storage.
func listFiles(t *testing.T, storage media.Storage, prefix string) []string {
	t.Helper()

	paths, err := storage.List(t.Context(), prefix)
	require.NoError(t, err)
	return paths
}

// zipContents reads every entry of a zip archive, keyed by its name.
func zipContents(t *testing.T, data []byte) map[string]string {
	t.Helper()

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)

	contents := map[string]string{}
	for _, file := range reader.File {
		rc, err := file.Open()
		require.NoError(t, err)
		content, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		contents[file.Name] = string(content)
	}
	return contents
}

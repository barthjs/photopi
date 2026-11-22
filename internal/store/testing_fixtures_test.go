package store

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
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

// newTestRunStore returns a RunStore on storage with an index of its own.
func newTestRunStore(storage media.Storage) *RunStore {
	return NewRunStore(storage, NewRunFileIndex(storage))
}

// newTestCaptureStore returns a CaptureStore on storage with an index of its own.
func newTestCaptureStore(storage media.Storage) *CaptureStore {
	return NewCaptureStore(storage, NewRunFileIndex(storage))
}

// putFile stores data at path in storage.
func putFile(t *testing.T, storage media.Storage, storagePath string, data []byte) {
	t.Helper()

	require.NoError(t, storage.Put(t.Context(), storagePath, bytes.NewReader(data), int64(len(data))))
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

// invalidFileNames returns names that would leave the folder of a store
// or whose file name with the extension would exceed the file name limit.
func invalidFileNames(extension string) []string {
	tooLong := strings.Repeat("a", maxFileNameBytes-len(extension)+1)
	return []string{"", " ", ".", "..", "../escape", "a/b", `a\b`, "/absolute", "a:b", "a?b", "a\x01b", "con", "NUL.backup", tooLong}
}

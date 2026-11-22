package store

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/barthjs/photopi/internal/media"
)

// runFilesTTL bounds how long a listing of the run files is used, so files changed in the storage from outside show up.
const runFilesTTL = 30 * time.Second

// RunFileIndex keeps the listings of the folders of the run files, which the stores sharing it drop on every change.
type RunFileIndex struct {
	storage media.Storage
	// generation counts the changes, so a listing that started before a change is never used after it.
	generation atomic.Uint64
	listings   singleflight.Group

	mu      sync.Mutex
	folders map[string]runFileListing
}

// runFileListing is the listing of one folder with the generation it started at.
type runFileListing struct {
	files      []runFile
	generation uint64
	listedAt   time.Time
}

// NewRunFileIndex creates a RunFileIndex. Every store that changes the run files on storage must share it.
func NewRunFileIndex(storage media.Storage) *RunFileIndex {
	return &RunFileIndex{
		storage: storage,
		folders: make(map[string]runFileListing),
	}
}

// list returns every run file below folder, see listRunFiles. Callers must not change the returned files.
func (i *RunFileIndex) list(ctx context.Context, folder string) ([]runFile, error) {
	generation := i.generation.Load()

	i.mu.Lock()
	listing, ok := i.folders[folder]
	i.mu.Unlock()
	if ok && listing.generation == generation && time.Since(listing.listedAt) < runFilesTTL {
		return listing.files, nil
	}

	result, err, _ := i.listings.Do(fmt.Sprintf("%s@%d", folder, generation), func() (any, error) {
		// Every caller that waits shares the listing, so it does not end with the request that started it
		files, err := listRunFiles(context.WithoutCancel(ctx), i.storage, folder)
		if err != nil {
			return nil, err
		}

		i.mu.Lock()
		defer i.mu.Unlock()
		if current, ok := i.folders[folder]; !ok || current.generation <= generation {
			i.folders[folder] = runFileListing{
				files:      files,
				generation: generation,
				listedAt:   time.Now(),
			}
		}
		return files, nil
	})
	if err != nil {
		return nil, err
	}

	files, _ := result.([]runFile)
	return files, nil
}

// invalidate drops every listing after a change of the run files.
func (i *RunFileIndex) invalidate() {
	i.generation.Add(1)
}

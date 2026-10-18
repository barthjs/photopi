package api_test

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/api"
	"github.com/barthjs/photopi/internal/apitest"
)

func TestGalleryAccess(t *testing.T) {
	t.Parallel()

	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/gallery",
			arrange: func(h *harness) (string, any) {
				completeGalleryRun(h)
				return "/api/gallery", nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodDelete,
			path:   "/api/gallery/{run}",
			arrange: func(h *harness) (string, any) {
				return galleryPath(completeGalleryRun(h)), nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/gallery/{run}/zip",
			arrange: func(h *harness) (string, any) {
				return galleryPath(completeGalleryRun(h)) + "/zip", nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/gallery/events",
			arrange: func(h *harness) (string, any) {
				completeGalleryRun(h)
				return "/api/gallery/events", nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/gallery/events/{event}/zip",
			arrange: func(h *harness) (string, any) {
				completeGalleryRun(h)
				events := h.asAdmin().GET("/api/gallery/events").AssertOK().Decode[[]api.GalleryEventResponse]()
				require.Len(h.t, events, 1)
				return "/api/gallery/events/" + events[0].Event + "/zip", nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/gallery/{run}/captures/{index}",
			arrange: func(h *harness) (string, any) {
				return galleryPath(completeGalleryRun(h)) + "/captures/1", nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/gallery/{run}/originals/{index}",
			arrange: func(h *harness) (string, any) {
				return galleryPath(completeGalleryRun(h)) + "/originals/1", nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
	}, withOverlayPath(), withOriginals())
}

// completeGalleryRun stores the camera overlay, completes a run, and returns its number.
func completeGalleryRun(h *harness) int {
	h.t.Helper()

	h.storeOverlay()
	return h.completeRun()
}

func TestGalleryListHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the runs", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		first := h.completeRun()
		second := h.completeRun()

		page := h.asAdmin().GET("/api/gallery").AssertOK().Decode[api.PaginatedResponse[api.GalleryRunResponse]]()

		assert.Equal(t, 2, page.Meta.TotalCount)
		require.Len(t, page.Data, 2)
		assert.Equal(t, second, page.Data[0].Run, "the newest run comes first")
		assert.Equal(t, first, page.Data[1].Run)
		assert.Equal(t, h.Config.Capture.ImagesPerRun, page.Data[0].ImageCount)
	})

	t.Run("counts the originals of a run", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withOverlayPath(), withOriginals())
		h.storeOverlay()
		run := h.startRun()
		h.uploadCapture(run, newJPEG(t, h.Faker)).AssertCreated()

		page := h.asAdmin().GET("/api/gallery").AssertOK().Decode[api.PaginatedResponse[api.GalleryRunResponse]]()

		require.Len(t, page.Data, 1)
		assert.Equal(t, 1, page.Data[0].OriginalCount)
	})

	t.Run("filters by event", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()
		client := h.asAdmin()
		events := client.GET("/api/gallery/events").AssertOK().Decode[[]api.GalleryEventResponse]()
		require.Len(t, events, 1)

		page := client.GET("/api/gallery?filter[event]=" + events[0].Event).AssertOK().Decode[api.PaginatedResponse[api.GalleryRunResponse]]()

		require.Len(t, page.Data, 1)
		assert.Equal(t, run, page.Data[0].Run)
		assert.Equal(t, events[0].Event, page.Data[0].Event)
	})

	t.Run("returns no runs for an unknown event", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.completeRun()

		response := h.asAdmin().GET("/api/gallery?filter[event]=" + h.Faker.LetterN(8))

		response.AssertOK().AssertJSONPath("meta.totalCount", 0)
	})

	t.Run("paginates", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		runs := h.Faker.IntRange(3, 5)
		for range runs {
			h.completeRun()
		}

		page := h.asAdmin().GET("/api/gallery?page[limit]=2&page[offset]=2").AssertOK().Decode[api.PaginatedResponse[api.GalleryRunResponse]]()

		assert.Equal(t, api.PaginatedMeta{
			TotalCount: runs,
			Limit:      2,
			Offset:     2,
		}, page.Meta)
		assert.Len(t, page.Data, min(2, runs-2))
	})

	t.Run("limits the page size", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().GET("/api/gallery?page[limit]=1000")

		response.AssertOK().AssertJSONPath("meta.limit", 100)
	})

	f := apitest.Faker(t)
	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "rejects a malformed page limit",
			query: "page[limit]=" + f.LetterN(3),
		},
		{
			name:  "rejects a page limit of zero",
			query: "page[limit]=0",
		},
		{
			name:  "rejects a malformed page offset",
			query: "page[offset]=" + f.LetterN(3),
		},
		{
			name:  "rejects a negative page offset",
			query: "page[offset]=-" + strconv.Itoa(f.IntRange(1, 100)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)

			response := h.asAdmin().GET("/api/gallery?" + tt.query)

			response.AssertBadRequest().AssertJSONPath("code", "bad_request")
		})
	}
}

func TestGalleryDeleteHandler(t *testing.T) {
	t.Parallel()

	t.Run("removes the run", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()
		client := h.asAdmin()

		response := client.DELETE(galleryPath(run))

		response.AssertNoContent()
		client.GET("/api/gallery").AssertJSONPath("meta.totalCount", 0)
		client.GET(galleryPath(run) + "/captures/1").AssertNotFound()
	})

	t.Run("rejects a malformed run", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().DELETE("/api/gallery/" + h.Faker.LetterN(3))

		response.AssertBadRequest()
	})
}

func TestGalleryZipHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the captures", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()

		response := h.asAdmin().GET(galleryPath(run) + "/zip")

		response.AssertOK().
			AssertHeader("Content-Type", "application/zip").
			AssertHeader("Content-Disposition", fmt.Sprintf("attachment; filename=run-%06d.zip", run))
		archive, err := zip.NewReader(bytes.NewReader(response.Body), int64(len(response.Body)))
		require.NoError(t, err)
		assert.Len(t, archive.File, h.Config.Capture.ImagesPerRun)
	})

	t.Run("returns the originals", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withOverlayPath(), withOriginals())
		h.storeOverlay()
		run := h.completeRun()

		response := h.asAdmin().GET(galleryPath(run) + "/zip?originals=true")

		response.AssertOK().
			AssertHeader("Content-Disposition", fmt.Sprintf("attachment; filename=run-%06d-originals.zip", run))
		archive, err := zip.NewReader(bytes.NewReader(response.Body), int64(len(response.Body)))
		require.NoError(t, err)
		assert.Len(t, archive.File, h.Config.Capture.ImagesPerRun)
	})

	t.Run("rejects a malformed run", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().GET("/api/gallery/" + h.Faker.LetterN(3) + "/zip")

		response.AssertBadRequest()
	})

	t.Run("rejects a malformed originals flag", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()

		response := h.asAdmin().GET(galleryPath(run) + "/zip?originals=" + h.Faker.LetterN(8))

		response.AssertBadRequest().AssertJSONPath("code", "bad_request")
	})

	t.Run("returns not found for an unknown run", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()

		response := h.asAdmin().GET(galleryPath(run+1) + "/zip")

		response.AssertNotFound().AssertJSONPath("code", "not_found")
	})

	t.Run("returns not found for a run without originals", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()

		response := h.asAdmin().GET(galleryPath(run) + "/zip?originals=true")

		response.AssertNotFound()
	})
}

func TestGalleryEventListHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the events", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.completeRun()

		events := h.asAdmin().GET("/api/gallery/events").AssertOK().Decode[[]api.GalleryEventResponse]()

		require.Len(t, events, 1)
		assert.True(t, events[0].Current, "the event of the booth is marked")
		assert.Equal(t, 1, events[0].RunCount)
		assert.Equal(t, h.Config.Capture.ImagesPerRun, events[0].ImageCount)
	})
}

func TestGalleryEventZipHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the captures of the event", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.completeRun()
		h.completeRun()
		client := h.asAdmin()
		events := client.GET("/api/gallery/events").AssertOK().Decode[[]api.GalleryEventResponse]()
		require.Len(t, events, 1)

		response := client.GET("/api/gallery/events/" + events[0].Event + "/zip")

		response.AssertOK().
			AssertHeader("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", events[0].Event))
		archive, err := zip.NewReader(bytes.NewReader(response.Body), int64(len(response.Body)))
		require.NoError(t, err)
		assert.Len(t, archive.File, 2*h.Config.Capture.ImagesPerRun)
	})

	t.Run("rejects a malformed originals flag", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.completeRun()
		client := h.asAdmin()
		events := client.GET("/api/gallery/events").AssertOK().Decode[[]api.GalleryEventResponse]()
		require.Len(t, events, 1)

		response := client.GET("/api/gallery/events/" + events[0].Event + "/zip?originals=" + h.Faker.LetterN(8))

		response.AssertBadRequest().AssertJSONPath("code", "bad_request")
	})

	t.Run("returns not found for an unknown event", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.completeRun()

		response := h.asAdmin().GET("/api/gallery/events/" + h.Faker.LetterN(4) + "..x/zip")

		response.AssertNotFound()
	})
}

func TestGalleryCaptureGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the capture", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.startRun()
		photo := newJPEG(t, h.Faker)
		h.uploadCapture(run, photo).AssertCreated()
		for range h.Config.Capture.ImagesPerRun - 1 {
			h.uploadCapture(run, newJPEG(t, h.Faker)).AssertCreated()
		}
		// A later run of the kiosk does not hide the earlier ones from the admin
		h.startRun()

		response := h.asAdmin().GET(galleryPath(run) + "/captures/1")

		response.AssertOK().AssertHeader("Content-Type", "image/jpeg")
		assert.Equal(t, photo, response.Body)
	})

	t.Run("rejects a malformed run", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().GET("/api/gallery/" + h.Faker.LetterN(3) + "/captures/1")

		response.AssertBadRequest()
	})

	t.Run("rejects a malformed index", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()

		response := h.asAdmin().GET(galleryPath(run) + "/captures/" + h.Faker.LetterN(3))

		response.AssertBadRequest()
	})

	t.Run("returns not found for an unknown capture", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()

		response := h.asAdmin().GET(galleryPath(run) + "/captures/" + strconv.Itoa(h.Config.Capture.ImagesPerRun+1))

		response.AssertNotFound()
	})
}

func TestGalleryOriginalGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the original", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withOverlayPath(), withOriginals())
		h.storeOverlay()
		run := h.startRun()
		photo := newJPEG(t, h.Faker)
		h.uploadCapture(run, photo).AssertCreated()
		client := h.asAdmin()

		response := client.GET(galleryPath(run) + "/originals/1")

		response.AssertOK()
		assert.Equal(t, photo, response.Body)
		composited := client.GET(galleryPath(run) + "/captures/1").AssertOK()
		assert.NotEqual(t, photo, composited.Body, "the capture carries the overlay")
	})

	t.Run("returns not found for an unknown original", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()

		response := h.asAdmin().GET(galleryPath(run) + "/originals/1")

		response.AssertNotFound()
	})
}

// withOriginals stores every capture with an overlay also without it.
func withOriginals() option {
	return func(h *harness) {
		h.Config.Capture.SaveOriginals = true
	}
}

// galleryPath returns the API path of a run in the gallery.
func galleryPath(run int) string {
	return "/api/gallery/" + strconv.Itoa(run)
}

package api_test

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image/jpeg"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

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
			method: http.MethodPost,
			path:   "/api/gallery/bulk",
			arrange: func(h *harness) (string, any) {
				return "/api/gallery/bulk", newGalleryBulkRequest("delete", strconv.Itoa(completeGalleryRun(h)))
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/gallery/zip",
			arrange: func(h *harness) (string, any) {
				return "/api/gallery/zip?filter[run]=" + strconv.Itoa(completeGalleryRun(h)), nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
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
			path:   "/api/gallery/days",
			arrange: func(h *harness) (string, any) {
				completeGalleryRun(h)
				return "/api/gallery/days", nil
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
		{
			method: http.MethodGet,
			path:   "/api/gallery/{run}/thumbnail",
			arrange: func(h *harness) (string, any) {
				return galleryPath(completeGalleryRun(h)) + "/thumbnail", nil
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
		require.NotNil(t, page.Data[0].TakenAt)
		takenAt, err := time.ParseInLocation("2006-01-02T15:04:05", *page.Data[0].TakenAt, time.Local)
		require.NoError(t, err, "the time is the wall time of the booth without a time zone")
		assert.WithinDuration(t, time.Now(), takenAt, time.Minute, "the time comes from the file names")
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

	t.Run("returns not found for an unknown run", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()

		response := h.asAdmin().DELETE(galleryPath(run + 1))

		response.AssertNotFound().AssertJSONPath("code", "not_found")
	})
}

func TestGalleryBulkHandler(t *testing.T) {
	t.Parallel()

	t.Run("deletes the runs", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		first := h.completeRun()
		second := h.completeRun()
		client := h.asAdmin()

		result := client.POST("/api/gallery/bulk", newGalleryBulkRequest("delete", strconv.Itoa(first), strconv.Itoa(second))).
			AssertOK().Decode[api.BulkResponse]()

		assert.Equal(t, api.BulkResponse{
			Succeeded: 2,
			FailedIDs: []string{},
		}, result)
		client.GET("/api/gallery").AssertJSONPath("meta.totalCount", 0)
	})

	t.Run("reports the failed ids", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()
		unknown := strconv.Itoa(run + 1)
		malformed := h.Faker.LetterN(8)
		client := h.asAdmin()

		result := client.POST("/api/gallery/bulk", newGalleryBulkRequest("delete", unknown, strconv.Itoa(run), malformed)).
			AssertOK().Decode[api.BulkResponse]()

		assert.Equal(t, 1, result.Succeeded, "the known run is still deleted")
		assert.Equal(t, 2, result.Failed)
		assert.Equal(t, []string{unknown, malformed}, result.FailedIDs)
		client.GET("/api/gallery").AssertJSONPath("meta.totalCount", 0)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		tooMany := make([]string, 201)
		for i := range tooMany {
			tooMany[i] = strconv.Itoa(i + 1)
		}
		duplicate := strconv.Itoa(f.IntRange(1, 1000))

		tests := []struct {
			name  string
			body  map[string]any
			field string
			tag   string
		}{
			{
				name: "missing action",
				body: map[string]any{
					"ids": []string{"1"},
				},
				field: "action",
				tag:   "required",
			},
			{
				name:  "unknown action",
				body:  newGalleryBulkRequest(f.LetterN(8), "1"),
				field: "action",
				tag:   "oneof",
			},
			{
				name: "missing ids",
				body: map[string]any{
					"action": "delete",
				},
				field: "ids",
				tag:   "required",
			},
			{
				name:  "no ids",
				body:  newGalleryBulkRequest("delete"),
				field: "ids",
				tag:   "min",
			},
			{
				name:  "too many ids",
				body:  newGalleryBulkRequest("delete", tooMany...),
				field: "ids",
				tag:   "max",
			},
			{
				name:  "duplicate ids",
				body:  newGalleryBulkRequest("delete", duplicate, duplicate),
				field: "ids",
				tag:   "unique",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := newHarness(t)

				response := h.asAdmin().POST("/api/gallery/bulk", tt.body)

				assertFieldError(t, response, tt.field, tt.tag)
			})
		}
	})
}

// newGalleryBulkRequest returns the body of a bulk action on the runs with the ids.
func newGalleryBulkRequest(action string, ids ...string) map[string]any {
	if ids == nil {
		ids = []string{}
	}
	return map[string]any{
		"action": action,
		"ids":    ids,
	}
}

func TestGalleryZipHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the captures of the runs", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		first := h.completeRun()
		second := h.completeRun()

		client := h.asAdmin()
		event := galleryEvent(client)

		response := client.GET(fmt.Sprintf("/api/gallery/zip?filter[run]=%d,%d", first, second))

		response.AssertOK().
			AssertHeader("Content-Type", "application/zip").
			AssertHeader("Content-Disposition", fmt.Sprintf("attachment; filename=%s-runs.zip", event))
		archive, err := zip.NewReader(bytes.NewReader(response.Body), int64(len(response.Body)))
		require.NoError(t, err)
		assert.Len(t, archive.File, 2*h.Config.Capture.ImagesPerRun)
	})

	t.Run("returns the originals", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withOverlayPath(), withOriginals())
		h.storeOverlay()
		run := h.completeRun()

		client := h.asAdmin()
		event := galleryEvent(client)

		response := client.GET(fmt.Sprintf("/api/gallery/zip?filter[run]=%d&originals=true", run))

		response.AssertOK().
			AssertHeader("Content-Disposition", fmt.Sprintf("attachment; filename=%s-runs-originals.zip", event))
		archive, err := zip.NewReader(bytes.NewReader(response.Body), int64(len(response.Body)))
		require.NoError(t, err)
		assert.Len(t, archive.File, h.Config.Capture.ImagesPerRun)
	})

	t.Run("names the archive without an event for runs of several events", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		require.NoError(t, h.Storage.Put(t.Context(), "captures/other-event/x_000050-01_20260101-000000.jpg", bytes.NewReader(newJPEG(t, h.Faker)), -1))
		run := h.completeRun()

		response := h.asAdmin().GET(fmt.Sprintf("/api/gallery/zip?filter[run]=50,%d", run))

		response.AssertOK().AssertHeader("Content-Disposition", "attachment; filename=runs.zip")
	})

	t.Run("leaves out unknown runs", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()

		response := h.asAdmin().GET(fmt.Sprintf("/api/gallery/zip?filter[run]=%d,%d", run, run+1))

		response.AssertOK()
		archive, err := zip.NewReader(bytes.NewReader(response.Body), int64(len(response.Body)))
		require.NoError(t, err)
		assert.Len(t, archive.File, h.Config.Capture.ImagesPerRun)
	})

	f := apitest.Faker(t)
	tooMany := make([]string, 201)
	for i := range tooMany {
		tooMany[i] = strconv.Itoa(i + 1)
	}
	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "rejects a missing run list",
			query: "",
		},
		{
			name:  "rejects an empty run list",
			query: "filter[run]=",
		},
		{
			name:  "rejects a malformed run",
			query: "filter[run]=1," + f.LetterN(3),
		},
		{
			name:  "rejects more runs than the list allows",
			query: "filter[run]=" + strings.Join(tooMany, ","),
		},
		{
			name:  "rejects a malformed originals flag",
			query: "filter[run]=1&originals=" + f.LetterN(8),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)

			response := h.asAdmin().GET("/api/gallery/zip?" + tt.query)

			response.AssertBadRequest().AssertJSONPath("code", "bad_request")
		})
	}

	t.Run("returns not found if no run has a file", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()

		response := h.asAdmin().GET(fmt.Sprintf("/api/gallery/zip?filter[run]=%d,%d", run+1, run+2))

		response.AssertNotFound().AssertJSONPath("code", "not_found")
	})
}

func TestGalleryRunZipHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the captures", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()

		client := h.asAdmin()
		event := galleryEvent(client)

		response := client.GET(galleryPath(run) + "/zip")

		response.AssertOK().
			AssertHeader("Content-Type", "application/zip").
			AssertHeader("Content-Disposition", fmt.Sprintf("attachment; filename=%s-run-%06d.zip", event, run))
		archive, err := zip.NewReader(bytes.NewReader(response.Body), int64(len(response.Body)))
		require.NoError(t, err)
		assert.Len(t, archive.File, h.Config.Capture.ImagesPerRun)
	})

	t.Run("returns the originals", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withOverlayPath(), withOriginals())
		h.storeOverlay()
		run := h.completeRun()

		client := h.asAdmin()
		event := galleryEvent(client)

		response := client.GET(galleryPath(run) + "/zip?originals=true")

		response.AssertOK().
			AssertHeader("Content-Disposition", fmt.Sprintf("attachment; filename=%s-run-%06d-originals.zip", event, run))
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

func TestGalleryDayListHandler(t *testing.T) {
	t.Parallel()

	t.Run("counts the runs per day", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.completeRun()
		h.completeRun()
		client := h.asAdmin()
		page := client.GET("/api/gallery").AssertOK().Decode[api.PaginatedResponse[api.GalleryRunResponse]]()
		require.Len(t, page.Data, 2)
		require.NotNil(t, page.Data[1].TakenAt)

		response := client.GET("/api/gallery/days").AssertOK().Decode[api.GalleryDayListResponse]()

		require.NotEmpty(t, response.Days)
		assert.Equal(t, (*page.Data[1].TakenAt)[:len(time.DateOnly)], response.Days[len(response.Days)-1].Day, "the days follow the wall time of the runs")
		total := 0
		for _, day := range response.Days {
			total += day.Count
		}
		assert.Equal(t, 2, total, "every run counts once")
	})

	t.Run("filters by event", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.completeRun()

		response := h.asAdmin().GET("/api/gallery/days?filter[event]=" + h.Faker.LetterN(8))

		response.AssertOK().AssertJSON(api.GalleryDayListResponse{
			Days: []api.GalleryDayResponse{},
		})
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

	t.Run("returns not found for a malformed event", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.completeRun()

		response := h.asAdmin().GET("/api/gallery/events/" + h.Faker.LetterN(4) + "..x/zip")

		response.AssertNotFound()
	})

	t.Run("returns not found for an unknown event", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.completeRun()

		response := h.asAdmin().GET("/api/gallery/events/" + strings.ToLower(h.Faker.LetterN(8)) + "/zip")

		response.AssertNotFound().AssertJSONPath("code", "not_found")
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

		response.AssertOK().
			AssertHeader("Content-Type", "image/jpeg").
			AssertHeader("Cache-Control", "private, no-cache")
		assert.Equal(t, photo, response.Body)
		assert.NotEmpty(t, response.Header.Get("ETag"))
	})

	t.Run("answers not modified for the image the browser holds", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()
		client := h.asAdmin()
		etag := client.GET(galleryPath(run) + "/captures/1").AssertOK().Header.Get("ETag")

		response := client.WithHeader("If-None-Match", etag).GET(galleryPath(run) + "/captures/1")

		response.AssertStatus(http.StatusNotModified)
		assert.Empty(t, response.Body)
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

func TestGalleryThumbnailGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the screen version of the first capture", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()

		response := h.asAdmin().GET(galleryPath(run) + "/thumbnail")

		response.AssertOK().
			AssertHeader("Content-Type", "image/jpeg").
			AssertHeader("Cache-Control", "private, no-cache")
		thumbnail, err := jpeg.Decode(bytes.NewReader(response.Body))
		require.NoError(t, err)
		assert.Equal(t, imageEdge, thumbnail.Bounds().Dx(), "a capture smaller than the screen keeps its size")
	})

	t.Run("returns the first capture of a run without a thumbnail", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		photo := newJPEG(t, h.Faker)
		require.NoError(t, h.Storage.Put(t.Context(), "captures/old-event/x_000050-01_20260101-000000.jpg", bytes.NewReader(photo), -1))

		response := h.asAdmin().GET(galleryPath(50) + "/thumbnail")

		response.AssertOK()
		assert.Equal(t, photo, response.Body)
	})

	t.Run("answers not modified for the thumbnail the browser holds", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()
		client := h.asAdmin()
		etag := client.GET(galleryPath(run) + "/thumbnail").AssertOK().Header.Get("ETag")

		response := client.WithHeader("If-None-Match", etag).GET(galleryPath(run) + "/thumbnail")

		response.AssertStatus(http.StatusNotModified)
	})

	t.Run("rejects a malformed run", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().GET("/api/gallery/" + h.Faker.LetterN(3) + "/thumbnail")

		response.AssertBadRequest()
	})

	t.Run("returns not found for an unknown run", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()

		response := h.asAdmin().GET(galleryPath(run+1) + "/thumbnail")

		response.AssertNotFound()
	})

	t.Run("returns not found after the run was deleted", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()
		client := h.asAdmin()
		client.GET(galleryPath(run) + "/thumbnail").AssertOK()
		client.DELETE(galleryPath(run)).AssertNoContent()

		response := client.GET(galleryPath(run) + "/thumbnail")

		response.AssertNotFound()
	})
}

// galleryEvent returns the slug of the only event of the gallery.
func galleryEvent(client *apitest.Client) string {
	page := client.GET("/api/gallery").AssertOK().Decode[api.PaginatedResponse[api.GalleryRunResponse]]()
	if len(page.Data) == 0 {
		return ""
	}
	return page.Data[0].Event
}

// withOriginals stores every capture with an overlay also without it.
func withOriginals() option {
	return func(h *harness) {
		h.Config.Capture.SaveOriginals = true
	}
}

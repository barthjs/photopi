package api_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/barthjs/photopi/internal/api"
)

func TestRunAccess(t *testing.T) {
	t.Parallel()

	// A run is only reachable from the client of the guest who started it
	testAccess(t, []accessRoute{
		{
			method: http.MethodPost,
			path:   "/api/camera/runs",
			arrange: func(h *harness) (string, any) {
				startRunWithCapture(h)
				return "/api/camera/runs", nil
			},
			statuses: map[*clientKind]int{
				clientOtherGuest: http.StatusConflict,
				clientGuest:      http.StatusCreated,
			},
		},
		{
			method: http.MethodDelete,
			path:   "/api/camera/runs/{run}",
			arrange: func(h *harness) (string, any) {
				return runPath(startRunWithCapture(h)), nil
			},
			statuses: map[*clientKind]int{
				clientOtherGuest: http.StatusNotFound,
				clientGuest:      http.StatusNoContent,
			},
		},
	})
}

// startRunWithCapture lets the guest of the harness start a run with one capture and returns its number.
func startRunWithCapture(h *harness) int {
	h.t.Helper()

	run := h.startRun()
	h.uploadCapture(run, newJPEG(h.t, h.Faker)).AssertCreated()
	return run
}

func TestRunCreateHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the run", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.POST("/api/camera/runs", nil)

		run := response.AssertCreated().Decode[api.RunResponse]()
		assert.Equal(t, 1, run.Run)
		assert.Equal(t, h.Config.Capture.ImagesPerRun, run.ImagesPerRun)
	})

	t.Run("returns the next run after a complete one", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		first := h.completeRun()

		response := h.Client.POST("/api/camera/runs", nil)

		second := response.AssertCreated().Decode[api.RunResponse]()
		assert.Equal(t, first+1, second.Run)
	})

	t.Run("discards the incomplete run of the same client", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		first := h.startRun()
		h.uploadCapture(first, newJPEG(t, h.Faker)).AssertCreated()

		second := h.startRun()

		assert.Equal(t, first+1, second, "run numbers are never handed out twice")
		h.asAdmin().GET("/api/gallery").AssertOK().AssertJSONPath("meta.totalCount", 0)
	})

	t.Run("rejects a run while the run of another client is open", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.startRun()

		response := h.newClient().POST("/api/camera/runs", nil)

		response.AssertStatus(http.StatusConflict).AssertJSONPath("code", "conflict")
	})
}

func TestRunAbortHandler(t *testing.T) {
	t.Parallel()

	t.Run("removes the run", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.startRun()
		h.uploadCapture(run, newJPEG(t, h.Faker)).AssertCreated()

		response := h.Client.DELETE(runPath(run))

		response.AssertNoContent()
		h.Client.GET(runPath(run) + "/captures/1").AssertNotFound()
		h.uploadCapture(run, newJPEG(t, h.Faker)).AssertStatus(http.StatusConflict)
	})
}

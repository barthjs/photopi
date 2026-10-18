package api_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/camera"
	"github.com/barthjs/photopi/internal/config"
)

func TestCameraAccess(t *testing.T) {
	t.Parallel()

	// The overlay is public, the preview only serves the client taking the photos of a run
	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/camera/overlay",
			arrange: func(h *harness) (string, any) {
				h.storeOverlay()
				h.startRun()
				return "/api/camera/overlay", nil
			},
			statuses: map[*clientKind]int{
				clientOtherGuest: http.StatusOK,
				clientGuest:      http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/camera/preview",
			arrange: func(h *harness) (string, any) {
				h.storeOverlay()
				h.startRun()
				return "/api/camera/preview", nil
			},
			statuses: map[*clientKind]int{
				clientOtherGuest: http.StatusNotFound,
				clientGuest:      http.StatusBadRequest,
			},
		},
	}, withOverlayPath())
}

func TestCameraOverlayGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the overlay", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withOverlayPath())
		overlay := h.storeOverlay()

		response := h.Client.GET("/api/camera/overlay")

		response.AssertOK().AssertHeader("Content-Type", "image/png")
		assert.Equal(t, overlay, response.Body)
	})

	t.Run("returns not found without an overlay", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.GET("/api/camera/overlay")

		response.AssertNotFound()
	})

	t.Run("returns not found for an unknown overlay", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withOverlayPath())

		response := h.Client.GET("/api/camera/overlay")

		response.AssertNotFound()
	})
}

func TestCameraPreviewGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("streams the device camera", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withDeviceCamera())
		srv, client := newPreviewServer(t, h)

		response, err := getPreview(t, srv, client)

		require.NoError(t, err)
		defer func() { _ = response.Body.Close() }()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.True(t, strings.HasPrefix(response.Header.Get("Content-Type"), "multipart/x-mixed-replace; boundary="))
		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		assert.True(t, bytes.HasPrefix(body, []byte("--photopiframe\r\n")))
		assert.True(t, bytes.Contains(body, h.Camera.image))
	})

	t.Run("rejects the webcam", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.startRun()

		response := h.Client.GET("/api/camera/preview")

		response.AssertBadRequest()
	})

	t.Run("returns not found without a run", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withDeviceCamera())

		response := h.Client.GET("/api/camera/preview")

		response.AssertNotFound()
	})

	t.Run("reports a failing camera", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withDeviceCamera())
		h.Camera.failPreview(errFake)
		srv, client := newPreviewServer(t, h)

		response, err := getPreview(t, srv, client)

		require.NoError(t, err)
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
		h.Client.GET("/readyz").AssertOK().AssertJSONPath("checks.camera", "unreachable")
	})

	t.Run("keeps the camera healthy when the client leaves", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withDeviceCamera())
		h.Camera.streamEndlessly()
		srv, client := newPreviewServer(t, h)

		response, err := getPreview(t, srv, client)

		require.NoError(t, err)
		_, err = io.ReadFull(response.Body, make([]byte, len(h.Camera.image)))
		require.NoError(t, err)
		_ = response.Body.Close()
		srv.CloseClientConnections()
		// The stream only ends once a write to the closed connection fails
		require.Eventually(t, func() bool { return h.Camera.endedPreviews() == 1 }, eventuallyTimeout, eventuallyTick)
		h.Client.GET("/readyz").AssertOK().AssertJSONPath("checks.camera", "ok")
	})
}

// withDeviceCamera selects a camera source that captures on the device, which the fake camera serves.
func withDeviceCamera() option {
	return withConfig(func(cfg *config.Config) { cfg.Camera.Source = camera.SourceRPiCam })
}

// newPreviewServer serves the application over real HTTP,
// since the stream sets write deadlines that only a real server supports,
// and returns a client whose session has opened a run, which the preview requires.
func newPreviewServer(t *testing.T, h *harness) (*httptest.Server, *http.Client) {
	t.Helper()

	srv := httptest.NewTestServer(t, h.handler)
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := srv.Client()
	client.Jar = jar

	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/api/camera/runs", nil)
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	require.NoError(t, err)
	_ = response.Body.Close()
	require.Equal(t, http.StatusCreated, response.StatusCode)

	return srv, client
}

// getPreview requests the preview stream with the client of newPreviewServer.
func getPreview(t *testing.T, srv *httptest.Server, client *http.Client) (*http.Response, error) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/api/camera/preview", nil)
	require.NoError(t, err)

	return client.Do(request)
}

package api_test

import (
	"bytes"
	"image/jpeg"
	"net/http"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/api"
	"github.com/barthjs/photopi/internal/apitest"
)

func TestCaptureAccess(t *testing.T) {
	t.Parallel()

	// A capture is only reachable from the client of the guest who started its run
	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/camera/runs/{run}/captures/{index}",
			arrange: func(h *harness) (string, any) {
				return runPath(startRunWithCapture(h)) + "/captures/1", nil
			},
			statuses: map[*clientKind]int{
				clientOtherGuest: http.StatusNotFound,
				clientGuest:      http.StatusOK,
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/camera/runs/{run}/captures",
			arrange: func(h *harness) (string, any) {
				return runPath(startRunWithCapture(h)) + "/captures", newUpload(h.Faker, "jpg", newJPEG(h.t, h.Faker))
			},
			statuses: map[*clientKind]int{
				clientOtherGuest: http.StatusNotFound,
				clientGuest:      http.StatusCreated,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/camera/runs/{run}/captures/{index}/preview",
			arrange: func(h *harness) (string, any) {
				return runPath(startRunWithCapture(h)) + "/captures/1/preview", nil
			},
			statuses: map[*clientKind]int{
				clientOtherGuest: http.StatusNotFound,
				clientGuest:      http.StatusOK,
			},
		},
	})
}

func TestCaptureGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the capture", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.startRun()
		photo := newJPEG(t, h.Faker)
		h.uploadCapture(run, photo).AssertCreated()

		response := h.Client.GET(runPath(run) + "/captures/1")

		response.AssertOK().AssertHeader("Content-Type", "image/jpeg")
		assert.Equal(t, photo, response.Body)
	})

	t.Run("rejects a malformed index", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.startRun()

		response := h.Client.GET(runPath(run) + "/captures/0")

		response.AssertBadRequest()
	})

	t.Run("returns not found for an unknown capture", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.startRun()

		response := h.Client.GET(runPath(run) + "/captures/1")

		response.AssertNotFound()
	})
}

func TestCaptureCreateHandler(t *testing.T) {
	t.Parallel()

	t.Run("stores the capture", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.startRun()
		photo := newJPEG(t, h.Faker)

		capture := h.uploadCapture(run, photo).AssertCreated().Decode[api.CaptureResponse]()

		assert.Equal(t, run, capture.Run)
		assert.Equal(t, 1, capture.ImageIndex)
		assert.Equal(t, h.Config.Capture.ImagesPerRun, capture.ImagesPerRun)
		stored := h.Client.GET(runPath(run)+"/captures/1").AssertOK().AssertHeader("Content-Type", "image/jpeg")
		assert.Equal(t, photo, stored.Body)
	})

	t.Run("takes the image with the device camera", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withDeviceCamera())
		run := h.startRun()

		response := h.Client.POST(runPath(run)+"/captures", nil)

		response.AssertCreated().AssertJSONPath("imageIndex", 1)
		assert.Equal(t, 1, h.Camera.captureCount())
		stored := h.Client.GET(runPath(run) + "/captures/1").AssertOK()
		assert.Equal(t, h.Camera.image, stored.Body)
	})

	t.Run("composites the overlay", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withOverlayPath())
		h.storeOverlay()
		run := h.startRun()
		photo := newJPEG(t, h.Faker)

		response := h.uploadCapture(run, photo)

		response.AssertCreated()
		stored := h.Client.GET(runPath(run) + "/captures/1").AssertOK()
		assert.NotEqual(t, photo, stored.Body)
		_, err := jpeg.Decode(bytes.NewReader(stored.Body))
		assert.NoError(t, err)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		pngData := newPNG(t, faker)
		oversizedData := newOversizedJPEG(t, faker)

		tests := []struct {
			name   string
			change func(*apitest.Multipart, *gofakeit.Faker)
			field  string
			tag    string
		}{
			{
				name: "png file",
				change: func(m *apitest.Multipart, f *gofakeit.Faker) {
					m.Files[0].Name = f.LetterN(10) + ".png"
					m.Files[0].Content = pngData
				},
				field: "file",
				tag:   "unsupported_type",
			},
			{
				name: "missing file",
				change: func(m *apitest.Multipart, f *gofakeit.Faker) {
					m.Files = nil
					m.Fields = map[string]string{"note": f.Word()}
				},
				field: "file",
				tag:   "required",
			},
			{
				name: "image above the pixel budget",
				change: func(m *apitest.Multipart, _ *gofakeit.Faker) {
					m.Files[0].Content = oversizedData
				},
				field: "file",
				tag:   "resolution_too_high",
			},
			{
				name: "undecodable image",
				change: func(m *apitest.Multipart, f *gofakeit.Faker) {
					// The JPEG signature passes the type check, the rest cannot be decoded
					m.Files[0].Content = append([]byte{0xFF, 0xD8, 0xFF}, f.LoremIpsumSentence(8)...)
				},
				field: "file",
				tag:   "invalid_format",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := newHarness(t)
				run := h.startRun()
				body := newUpload(h.Faker, "jpg", newJPEG(t, h.Faker))
				tt.change(&body, h.Faker)

				response := h.Client.POST(runPath(run)+"/captures", body)

				assertFieldError(t, response, tt.field, tt.tag)
				h.Client.GET(runPath(run) + "/captures/1").AssertNotFound()
			})
		}
	})

	t.Run("rejects a malformed run", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.Client.POST("/api/camera/runs/"+h.Faker.LetterN(4)+"/captures", nil)

		response.AssertBadRequest()
	})

	t.Run("rejects a closed run", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()

		response := h.uploadCapture(run, newJPEG(t, h.Faker))

		response.AssertStatus(http.StatusConflict).AssertJSONPath("code", "conflict")
	})

	t.Run("reports a failing camera", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withDeviceCamera())
		h.Camera.fail(errFake)
		run := h.startRun()

		response := h.Client.POST(runPath(run)+"/captures", nil)

		response.AssertStatus(http.StatusInternalServerError)
		// The gallery and shared links keep working without the camera, so the booth reports itself degraded
		h.Client.GET("/readyz").AssertOK().AssertJSONPath("status", "degraded").AssertJSONPath("checks.camera", "unreachable")
	})

	t.Run("reports an unusable image", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withDeviceCamera())
		photo := newJPEG(t, h.Faker)
		h.Camera.returnImage(photo[:len(photo)/2])
		run := h.startRun()

		response := h.Client.POST(runPath(run)+"/captures", nil)

		response.AssertStatus(http.StatusBadGateway).AssertJSONPath("code", "camera_capture_failed")
		h.Client.GET("/readyz").AssertOK().AssertJSONPath("checks.camera", "unreachable")
	})
}

func TestCapturePreviewGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the screen version", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.startRun()
		h.uploadCapture(run, newJPEG(t, h.Faker)).AssertCreated()

		response := h.Client.GET(runPath(run) + "/captures/1/preview")

		response.AssertOK().AssertHeader("Content-Type", "image/jpeg")
		img, err := jpeg.Decode(bytes.NewReader(response.Body))
		require.NoError(t, err)
		assert.LessOrEqual(t, img.Bounds().Dx(), 800, "the screen version fits the kiosk")
	})

	t.Run("rejects a malformed index", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.startRun()

		response := h.Client.GET(runPath(run) + "/captures/" + h.Faker.LetterN(3) + "/preview")

		response.AssertBadRequest()
	})

	t.Run("returns not found for an unknown capture", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.startRun()
		h.uploadCapture(run, newJPEG(t, h.Faker)).AssertCreated()

		response := h.Client.GET(runPath(run) + "/captures/2/preview")

		response.AssertNotFound()
	})
}

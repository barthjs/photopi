package api_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"strconv"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/api"
	"github.com/barthjs/photopi/internal/apitest"
	"github.com/barthjs/photopi/internal/config"
	"github.com/barthjs/photopi/internal/media"
)

// newAuthLoginRequest returns a login with a random password, which never matches the admin password.
func newAuthLoginRequest(f *gofakeit.Faker) api.AuthLoginRequest {
	return api.AuthLoginRequest{
		Password: f.Password(true, true, true, false, false, 20),
	}
}

// newConfigRequest returns a valid save request for cfg with random values.
func newConfigRequest(f *gofakeit.Faker, cfg config.Config) api.ConfigSaveRequest {
	return api.ConfigSaveRequest{
		App: api.AppSettings{
			Name:     f.AppName(),
			Env:      string(cfg.App.Env),
			Secret:   f.Password(true, true, true, false, false, 16),
			Language: string(cfg.App.Language),
		},
		Log: api.LogSettings{
			Level:   cfg.Log.RawLevel,
			Channel: string(cfg.Log.Channel),
			Format:  string(cfg.Log.Format),
		},
		Camera: api.CameraSettings{
			Source:        string(cfg.Camera.Source),
			ExposureTime:  string(cfg.Camera.ExposureTime),
			ExposureMode:  string(cfg.Camera.ExposureMode),
			WhiteBalance:  string(cfg.Camera.WhiteBalance),
			Rotation:      cfg.Camera.Rotation,
			JPEGQuality:   f.IntRange(1, 100),
			AutofocusMode: string(cfg.Camera.AutofocusMode),
			Denoise:       string(cfg.Camera.Denoise),
		},
		Capture: api.CaptureSettings{
			ImagesPerRun:   f.IntRange(1, 20),
			FilePrefix:     f.LetterN(8),
			EventName:      f.Company(),
			WelcomeMessage: f.LetterN(20),
			AdminEmail:     f.Email(),
		},
		Print: api.PrintSettings{
			SubmissionMode: cfg.Print.SubmissionMode,
		},
		Mail: api.MailSettings{
			Driver: string(cfg.Mail.Driver),
			Port:   cfg.Mail.Port,
		},
		Server: api.ServerSettings{
			Host:               cfg.Server.Host,
			Port:               f.IntRange(1024, 65535),
			CORSAllowedOrigins: cfg.Server.CORSAllowedOrigins,
		},
		Storage: api.StorageSettings{
			Driver: string(cfg.Storage.Driver),
			Path:   cfg.Storage.Path,
		},
		Theme: api.ThemeSettings{
			Primary: f.HexColor(),
		},
	}
}

// newS3StorageSettings returns storage settings of the s3 driver with every field the driver needs.
func newS3StorageSettings(f *gofakeit.Faker) api.StorageSettings {
	return api.StorageSettings{
		Driver: string(media.DriverS3),
		S3: api.S3Settings{
			Endpoint:  f.URL(),
			Bucket:    f.LetterN(10),
			AccessKey: f.LetterN(16),
			SecretKey: f.LetterN(32),
		},
	}
}

// newLayoutSaveRequest returns a valid print layout with random slots.
func newLayoutSaveRequest(f *gofakeit.Faker, opts ...func(*api.LayoutSaveRequest)) api.LayoutSaveRequest {
	slots := make([]api.LayoutSlotRequest, f.IntRange(1, 4))
	for i := range slots {
		w := f.Float64Range(0.1, 0.5)
		h := f.Float64Range(0.1, 0.5)
		slots[i] = api.LayoutSlotRequest{
			X:      f.Float64Range(0, 1-w),
			Y:      f.Float64Range(0, 1-h),
			W:      w,
			H:      h,
			Locked: f.Bool(),
		}
	}

	layout := api.LayoutSaveRequest{
		Name: f.ProductName(),
		Paper: api.LayoutPaperRequest{
			WidthMM:  f.Float64Range(50, 150),
			HeightMM: f.Float64Range(50, 150),
		},
		Pages: []api.LayoutPageRequest{
			{
				Slots: slots,
			},
		},
	}
	for _, opt := range opts {
		opt(&layout)
	}
	return layout
}

// completeRun opens a run, fills it with captures, and returns its number.
func (h *harness) completeRun() int {
	h.t.Helper()

	run := h.startRun()
	for range h.Config.Capture.ImagesPerRun {
		h.uploadCapture(run, newJPEG(h.t, h.Faker)).AssertCreated()
	}
	return run
}

// startRun opens a new run and returns its number.
func (h *harness) startRun() int {
	h.t.Helper()

	start := h.Client.POST("/api/camera/runs", nil).AssertCreated().Decode[api.RunResponse]()
	return start.Run
}

// uploadCapture stores data as the next capture of run, as the web interface does for a webcam.
func (h *harness) uploadCapture(run int, data []byte) *apitest.Response {
	h.t.Helper()
	return h.Client.POST(runPath(run)+"/captures", newUpload(h.Faker, "jpg", data))
}

// newUpload returns a multipart body with data as the "file" field.
func newUpload(f *gofakeit.Faker, extension string, data []byte) apitest.Multipart {
	return apitest.Multipart{
		Files: []apitest.File{
			{
				Field:   "file",
				Name:    f.LetterN(10) + "." + extension,
				Content: data,
			},
		},
	}
}

// withOverlayPath configures the camera overlay at a random path, which storeOverlay fills.
func withOverlayPath() option {
	return func(h *harness) {
		h.Config.Camera.OverlayPath = "overlays/" + h.Faker.LetterN(8) + ".png"
	}
}

// storeOverlay stores a random PNG as the configured camera overlay and returns it.
func (h *harness) storeOverlay() []byte {
	h.t.Helper()

	overlay := newPNG(h.t, h.Faker)
	err := h.Storage.Put(h.t.Context(), h.Config.Camera.OverlayPath, bytes.NewReader(overlay), int64(len(overlay)))
	require.NoError(h.t, err)
	return overlay
}

// imageEdge is the edge length of generated images, small enough to keep the tests fast.
const imageEdge = 32

// newPNG returns a random square PNG with an edge length of imageEdge.
func newPNG(t testing.TB, f *gofakeit.Faker) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, newPicture(f, imageEdge, imageEdge)))
	return buf.Bytes()
}

// newJPEG returns a random square JPEG with an edge length of imageEdge.
func newJPEG(t testing.TB, f *gofakeit.Faker) []byte {
	t.Helper()
	return encodeJPEG(t, newPicture(f, imageEdge, imageEdge))
}

// encodeJPEG encodes an image as JPEG.
func encodeJPEG(t testing.TB, img image.Image) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{
		Quality: 90,
	}))
	return buf.Bytes()
}

// newPicture draws a gradient with random blocks of the given size, so every generated image differs.
func newPicture(f *gofakeit.Faker, width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	from := color.RGBA{
		R: f.Uint8(),
		G: f.Uint8(),
		B: f.Uint8(),
		A: 255,
	}
	to := color.RGBA{
		R: f.Uint8(),
		G: f.Uint8(),
		B: f.Uint8(),
		A: 255,
	}

	for y := range height {
		share := float64(y) / float64(height)
		row := color.RGBA{
			R: mixChannel(from.R, to.R, share),
			G: mixChannel(from.G, to.G, share),
			B: mixChannel(from.B, to.B, share),
			A: 255,
		}
		for x := range width {
			img.SetRGBA(x, y, row)
		}
	}

	for range 12 {
		block := color.RGBA{
			R: f.Uint8(),
			G: f.Uint8(),
			B: f.Uint8(),
			A: 255,
		}
		x0, y0 := f.IntRange(0, width-1), f.IntRange(0, height-1)
		x1, y1 := min(width, x0+f.IntRange(width/8, width/2)), min(height, y0+f.IntRange(height/8, height/2))
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				img.SetRGBA(x, y, block)
			}
		}
	}

	return img
}

// mixChannel interpolates between two color channel values by share.
func mixChannel(a, b uint8, share float64) uint8 {
	return uint8(math.Round(float64(a) + (float64(b)-float64(a))*share))
}

// runPath returns the API path of a run.
func runPath(run int) string {
	return "/api/camera/runs/" + strconv.Itoa(run)
}

// sharePath returns the API path of the share of a run.
func sharePath(run int) string {
	return "/api/share/" + strconv.Itoa(run)
}

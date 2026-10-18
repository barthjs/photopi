package api_test

import (
	"bytes"
	"net/http"
	"net/url"
	"path"
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/api"
	"github.com/barthjs/photopi/internal/apitest"
	"github.com/barthjs/photopi/internal/media"
)

func TestOverlayAccess(t *testing.T) {
	t.Parallel()

	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/overlays",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/overlays",
			arrange: func(h *harness) (string, any) {
				return "/api/overlays", newUpload(h.Faker, "png", newPNG(h.t, h.Faker))
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusCreated,
			},
		},
		{
			method: http.MethodDelete,
			path:   "/api/overlays/{name}",
			arrange: func(h *harness) (string, any) {
				created := h.asAdmin().POST("/api/overlays", newUpload(h.Faker, "png", newPNG(h.t, h.Faker))).AssertCreated().Decode[api.OverlayResponse]()
				return overlayPath(created.Name), nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/overlays/{name}",
			arrange: func(h *harness) (string, any) {
				created := h.asAdmin().POST("/api/overlays", newUpload(h.Faker, "png", newPNG(h.t, h.Faker))).AssertCreated().Decode[api.OverlayResponse]()
				return overlayPath(created.Name), nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
	})
}

func TestOverlayListHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the overlays", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()
		created := client.POST("/api/overlays", newUpload(h.Faker, "png", newPNG(t, h.Faker))).AssertCreated().Decode[api.OverlayResponse]()

		response := client.GET("/api/overlays")

		response.AssertOK().AssertJSON([]api.OverlayResponse{created})
	})
}

func TestOverlayCreateHandler(t *testing.T) {
	t.Parallel()

	t.Run("stores the overlay", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()
		name := h.Faker.LetterN(10) + ".png"
		overlay := newPNG(t, h.Faker)

		created := client.POST("/api/overlays", apitest.Multipart{
			Files: []apitest.File{
				{
					Field:   "file",
					Name:    name,
					Content: overlay,
				},
			},
		}).AssertCreated().Decode[api.OverlayResponse]()

		assert.Equal(t, name, created.Name)
		assert.Equal(t, "overlays/"+name, created.Path)
		stored, err := media.ReadAll(t.Context(), h.Storage, created.Path)
		require.NoError(t, err)
		assert.Equal(t, overlay, stored)
		client.GET("/api/overlays").AssertOK().AssertJSON([]api.OverlayResponse{created})
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		jpegData := newJPEG(t, faker)

		tests := []struct {
			name   string
			change func(*apitest.Multipart, *gofakeit.Faker)
			field  string
			tag    string
		}{
			{
				name: "jpeg file",
				change: func(m *apitest.Multipart, f *gofakeit.Faker) {
					m.Files[0].Name = f.LetterN(10) + ".jpg"
					m.Files[0].Content = jpegData
				},
				field: "file",
				tag:   "unsupported_type",
			},
			{
				name: "missing file",
				change: func(m *apitest.Multipart, f *gofakeit.Faker) {
					m.Files = nil
					m.Fields = map[string]string{"name": f.Word()}
				},
				field: "file",
				tag:   "required",
			},
			{
				name: "too large file",
				change: func(m *apitest.Multipart, _ *gofakeit.Faker) {
					m.Files[0].Content = bytes.Repeat([]byte{0}, 11<<20)
				},
				field: "file",
				tag:   "too_large",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := newHarness(t)
				client := h.asAdmin()
				body := newUpload(h.Faker, "png", newPNG(t, h.Faker))
				tt.change(&body, h.Faker)

				response := client.POST("/api/overlays", body)

				assertFieldError(t, response, tt.field, tt.tag)
				client.GET("/api/overlays").AssertOK().AssertJSONCount("", 0)
			})
		}
	})

	t.Run("rejects a malformed body", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name        string
			contentType func(boundary string) string
			body        func(h *harness, boundary string) string
		}{
			{
				name:        "not multipart",
				contentType: func(string) string { return "application/json" },
				body: func(h *harness, _ string) string {
					return `{"file":"` + h.Faker.Word() + `"}`
				},
			},
			{
				name:        "missing boundary",
				contentType: func(string) string { return "multipart/form-data" },
				body: func(h *harness, _ string) string {
					return h.Faker.LoremIpsumSentence(4)
				},
			},
			{
				name:        "unterminated part",
				contentType: func(boundary string) string { return "multipart/form-data; boundary=" + boundary },
				body: func(h *harness, boundary string) string {
					return "--" + boundary + "\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\n" + h.Faker.Word()
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := newHarness(t)
				boundary := h.Faker.LetterN(16)
				client := h.asAdmin().WithHeader("Content-Type", tt.contentType(boundary))

				response := client.POST("/api/overlays", strings.NewReader(tt.body(h, boundary)))

				response.AssertBadRequest().AssertJSONPath("code", "bad_request")
			})
		}
	})
}

func TestOverlayDeleteHandler(t *testing.T) {
	t.Parallel()

	t.Run("removes the overlay", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()
		created := client.POST("/api/overlays", newUpload(h.Faker, "png", newPNG(t, h.Faker))).AssertCreated().Decode[api.OverlayResponse]()

		response := client.DELETE(overlayPath(created.Name))

		response.AssertNoContent()
		client.GET("/api/overlays").AssertOK().AssertJSONCount("", 0)
		_, err := media.ReadAll(t.Context(), h.Storage, created.Path)
		assert.ErrorIs(t, err, media.ErrNotFound)
	})

	t.Run("returns not found for an unknown overlay", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().DELETE(overlayPath(h.Faker.LetterN(8) + ".png"))

		response.AssertNotFound().AssertJSONPath("code", "not_found")
	})

	t.Run("returns not found for a malformed name", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()
		created := client.POST("/api/overlays", newUpload(h.Faker, "png", newPNG(t, h.Faker))).AssertCreated().Decode[api.OverlayResponse]()

		response := client.DELETE(overlayPath(".."))

		response.AssertNotFound().AssertJSONPath("code", "not_found")
		client.GET("/api/overlays").AssertOK().AssertJSON([]api.OverlayResponse{created})
	})

	t.Run("keeps the active overlay", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withOverlayPath())
		overlay := h.storeOverlay()

		response := h.asAdmin().DELETE(overlayPath(path.Base(h.Config.Camera.OverlayPath)))

		response.AssertStatus(http.StatusConflict).AssertJSONPath("code", "conflict")
		stored, err := media.ReadAll(t.Context(), h.Storage, h.Config.Camera.OverlayPath)
		require.NoError(t, err)
		assert.Equal(t, overlay, stored)
	})

	t.Run("keeps the overlay for a guest", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		created := h.asAdmin().POST("/api/overlays", newUpload(h.Faker, "png", newPNG(t, h.Faker))).AssertCreated().Decode[api.OverlayResponse]()

		h.Client.DELETE(overlayPath(created.Name)).AssertUnauthorized()

		h.asAdmin().GET("/api/overlays").AssertOK().AssertJSON([]api.OverlayResponse{created})
	})
}

func TestOverlayDownloadHandler(t *testing.T) {
	t.Parallel()

	t.Run("downloads the overlay as it is stored", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()
		overlay := newPNG(t, h.Faker)
		created := client.POST("/api/overlays", newUpload(h.Faker, "png", overlay)).AssertCreated().Decode[api.OverlayResponse]()

		response := client.GET(overlayPath(created.Name))

		response.AssertOK().
			AssertHeader("Content-Type", "image/png").
			AssertHeader("Content-Disposition", "attachment; filename="+created.Name)
		assert.Equal(t, overlay, response.Body)
	})

	t.Run("returns not found for an unknown overlay", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().GET(overlayPath(h.Faker.LetterN(8) + ".png"))

		response.AssertNotFound().AssertJSONPath("code", "not_found")
	})
}

// overlayPath returns the API path of an overlay.
func overlayPath(name string) string {
	return "/api/overlays/" + url.PathEscape(name)
}

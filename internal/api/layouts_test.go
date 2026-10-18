package api_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"

	"github.com/barthjs/photopi/internal/api"
)

// defaultLayoutIDs are the print layouts every installation starts with.
var defaultLayoutIDs = []string{"single-4x6", "strip-3"}

func TestLayoutAccess(t *testing.T) {
	t.Parallel()

	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/print/layouts",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/print/layouts/{id}",
			arrange: func(*harness) (string, any) {
				return layoutPath(defaultLayoutIDs[0]), nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodPut,
			path:   "/api/print/layouts/{id}",
			arrange: func(h *harness) (string, any) {
				return layoutPath(h.Faker.LetterN(8)), newLayoutSaveRequest(h.Faker)
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodDelete,
			path:   "/api/print/layouts/{id}",
			arrange: func(*harness) (string, any) {
				return layoutPath(defaultLayoutIDs[0]), nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
		},
	})
}

func TestLayoutListHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the layouts", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		layouts := h.asAdmin().GET("/api/print/layouts").AssertOK().Decode[[]api.LayoutResponse]()

		ids := make([]string, 0, len(layouts))
		for _, layout := range layouts {
			ids = append(ids, layout.ID)
			assert.Positive(t, layout.Slots)
			assert.Positive(t, layout.Pages)
		}
		assert.Equal(t, defaultLayoutIDs, ids)
	})
}

func TestLayoutGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the layout", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().GET(layoutPath(defaultLayoutIDs[0]))

		response.AssertOK().AssertJSONPath("id", defaultLayoutIDs[0]).AssertJSONPath("pages.0.slots.0.locked", false)
	})

	t.Run("returns not found for an unknown layout", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().GET(layoutPath(h.Faker.LetterN(10)))

		response.AssertNotFound().AssertJSONPath("code", "not_found")
	})

	t.Run("returns not found for a malformed id", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().GET(layoutPath(".."))

		response.AssertNotFound().AssertJSONPath("code", "not_found")
	})
}

func TestLayoutSaveHandler(t *testing.T) {
	t.Parallel()

	t.Run("stores the layout", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()
		id := h.Faker.LetterN(10)
		request := newLayoutSaveRequest(h.Faker)

		saved := client.PUT(layoutPath(id), request).AssertOK().Decode[api.LayoutDetailResponse]()

		assert.Equal(t, id, saved.ID)
		assert.Equal(t, request.Name, saved.Name)
		assert.Len(t, saved.Pages[0].Slots, len(request.Pages[0].Slots))
		client.GET(layoutPath(id)).AssertOK().AssertJSON(saved)
		client.GET("/api/print/layouts").AssertJSONCount("", len(defaultLayoutIDs)+1)
	})

	t.Run("replaces the layout", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()
		id := h.Faker.LetterN(10)
		client.PUT(layoutPath(id), newLayoutSaveRequest(h.Faker)).AssertOK()
		replaced := newLayoutSaveRequest(h.Faker)

		saved := client.PUT(layoutPath(id), replaced).AssertOK().Decode[api.LayoutDetailResponse]()

		assert.Equal(t, replaced.Name, saved.Name)
		client.GET(layoutPath(id)).AssertOK().AssertJSON(saved)
		client.GET("/api/print/layouts").AssertJSONCount("", len(defaultLayoutIDs)+1)
	})

	t.Run("stores the branding color", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		hexColor := h.Faker.HexColor()
		request := newLayoutSaveRequest(h.Faker, func(r *api.LayoutSaveRequest) {
			r.Branding.Color = hexColor
		})

		response := h.asAdmin().PUT(layoutPath(h.Faker.LetterN(10)), request)

		response.AssertOK().AssertJSONPath("branding.color", hexColor)
	})

	t.Run("stores the slot lock", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()
		id := h.Faker.LetterN(10)
		request := newLayoutSaveRequest(h.Faker, func(r *api.LayoutSaveRequest) {
			r.Pages[0].Slots[0].Locked = true
		})

		response := client.PUT(layoutPath(id), request)

		response.AssertOK().AssertJSONPath("pages.0.slots.0.locked", true)
		client.GET(layoutPath(id)).AssertOK().AssertJSONPath("pages.0.slots.0.locked", true)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		// The id is no field of the request but the file name of the layout, so its rows set the path
		tests := []struct {
			name   string
			change func(*api.LayoutSaveRequest, *gofakeit.Faker)
			field  string
			tag    string
			id     string
		}{
			{
				name:   "missing name",
				change: func(r *api.LayoutSaveRequest, _ *gofakeit.Faker) { r.Name = "" },
				field:  "name",
				tag:    "required",
			},
			{
				name: "paper wider than the limit",
				change: func(r *api.LayoutSaveRequest, f *gofakeit.Faker) {
					r.Paper.WidthMM = f.Float64Range(501, 1000)
				},
				field: "paper.widthMM",
				tag:   "max",
			},
			{
				name:   "missing pages",
				change: func(r *api.LayoutSaveRequest, _ *gofakeit.Faker) { r.Pages = nil },
				field:  "pages",
				tag:    "required",
			},
			{
				name:   "short branding color",
				change: func(r *api.LayoutSaveRequest, _ *gofakeit.Faker) { r.Branding.Color = "#fff" },
				field:  "branding.color",
				tag:    "rgbcolor",
			},
			{
				name:  "blank id",
				field: "id",
				tag:   "invalid",
				id:    " ",
			},
			{
				name:  "id outside the folder",
				field: "id",
				tag:   "invalid",
				id:    "..",
			},
			{
				name:  "id too long for a file",
				field: "id",
				tag:   "invalid",
				id:    strings.Repeat("a", 251),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := newHarness(t)
				client := h.asAdmin()
				id := tt.id
				if id == "" {
					id = h.Faker.LetterN(10)
				}
				request := newLayoutSaveRequest(h.Faker)
				if tt.change != nil {
					tt.change(&request, h.Faker)
				}

				response := client.PUT(layoutPath(id), request)

				assertFieldError(t, response, tt.field, tt.tag)
				client.GET("/api/print/layouts").AssertJSONCount("", len(defaultLayoutIDs))
			})
		}
	})

	t.Run("rejects a slot outside the page", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			slot api.LayoutSlotRequest
		}{
			{
				name: "past the right edge",
				slot: api.LayoutSlotRequest{
					X: 0.8,
					Y: 0.1,
					W: 0.5,
					H: 0.5,
				},
			},
			{
				name: "past the bottom edge",
				slot: api.LayoutSlotRequest{
					X: 0.1,
					Y: 0.8,
					W: 0.5,
					H: 0.5,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := newHarness(t)
				client := h.asAdmin()
				id := h.Faker.LetterN(10)
				request := newLayoutSaveRequest(h.Faker, func(r *api.LayoutSaveRequest) {
					r.Pages[0].Slots[0] = tt.slot
				})

				response := client.PUT(layoutPath(id), request)

				response.AssertBadRequest()
				client.GET(layoutPath(id)).AssertNotFound()
			})
		}
	})

	t.Run("stores nothing for a guest", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		id := h.Faker.LetterN(10)

		h.Client.PUT(layoutPath(id), newLayoutSaveRequest(h.Faker)).AssertUnauthorized()

		h.asAdmin().GET(layoutPath(id)).AssertNotFound()
	})
}

func TestLayoutDeleteHandler(t *testing.T) {
	t.Parallel()

	t.Run("removes the layout", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()

		response := client.DELETE(layoutPath(defaultLayoutIDs[0]))

		response.AssertNoContent()
		client.GET(layoutPath(defaultLayoutIDs[0])).AssertNotFound()
		client.GET("/api/print/layouts").AssertJSONCount("", len(defaultLayoutIDs)-1)
	})

	t.Run("returns not found for an unknown layout", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().DELETE(layoutPath(h.Faker.LetterN(8)))

		response.AssertNotFound().AssertJSONPath("code", "not_found")
	})

	t.Run("returns not found for a malformed id", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()

		response := client.DELETE(layoutPath(".."))

		response.AssertNotFound().AssertJSONPath("code", "not_found")
		client.GET("/api/print/layouts").AssertJSONCount("", len(defaultLayoutIDs))
	})

	t.Run("keeps the configured layout", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()

		response := client.DELETE(layoutPath(h.Config.Print.LayoutID))

		response.AssertStatus(http.StatusConflict).AssertJSONPath("code", "conflict")
		client.GET(layoutPath(h.Config.Print.LayoutID)).AssertOK()
	})

	t.Run("keeps the layout for a guest", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.Client.DELETE(layoutPath(defaultLayoutIDs[0])).AssertUnauthorized()

		h.asAdmin().GET(layoutPath(defaultLayoutIDs[0])).AssertOK()
	})
}

// layoutPath returns the API path of a print layout.
func layoutPath(id string) string {
	return "/api/print/layouts/" + url.PathEscape(id)
}

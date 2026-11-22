package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/api"
)

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
			path:   "/api/print/layouts/{name}",
			arrange: func(*harness) (string, any) {
				return layoutPath(defaultLayoutNames[0]), nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodPut,
			path:   "/api/print/layouts/{name}",
			arrange: func(h *harness) (string, any) {
				request := newLayoutSaveRequest(h.Faker)
				return layoutPath(request.Name), request
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodDelete,
			path:   "/api/print/layouts/{name}",
			arrange: func(*harness) (string, any) {
				return layoutPath(defaultLayoutNames[0]), nil
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

	t.Run("returns the layouts sorted by name and marks the one in use", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withPrinting("managed"))

		layouts := h.asAdmin().GET("/api/print/layouts").AssertOK().Decode[[]api.LayoutResponse]()

		names := make([]string, 0, len(layouts))
		inUse := make([]string, 0, 1)
		for _, layout := range layouts {
			names = append(names, layout.Name)
			if layout.InUse {
				inUse = append(inUse, layout.Name)
			}
			assert.Positive(t, layout.Slots)
			assert.Positive(t, layout.Pages)
		}
		assert.Equal(t, defaultLayoutNames, names)
		assert.Equal(t, []string{h.Config.Print.Layout}, inUse)
	})
}

func TestLayoutGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the layout", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().GET(layoutPath(defaultLayoutNames[0]))

		response.AssertOK().AssertJSONPath("name", defaultLayoutNames[0]).AssertJSONPath("pages.0.slots.0.locked", false)
	})

	t.Run("returns not found for an unknown layout", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().GET(layoutPath(layoutName(h.Faker)))

		response.AssertNotFound().AssertJSONPath("code", "not_found")
	})

	t.Run("returns not found for a malformed name", func(t *testing.T) {
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
		hexColor := h.Faker.HexColor()
		request := newLayoutSaveRequest(h.Faker, func(r *api.LayoutSaveRequest) {
			r.Branding.Color = hexColor
			r.Pages[0].Slots[0].Locked = true
		})

		saved := client.PUT(layoutPath(request.Name), request).AssertOK().Decode[api.LayoutDetailResponse]()

		assert.Equal(t, request.Name, saved.Name)
		assert.Equal(t, request.Description, saved.Description)
		assert.Equal(t, hexColor, saved.Branding.Color)
		assert.Len(t, saved.Pages[0].Slots, len(request.Pages[0].Slots))
		assert.True(t, saved.Pages[0].Slots[0].Locked)
		exists, err := h.Storage.Exists(t.Context(), "layouts/"+request.Name+".json")
		require.NoError(t, err)
		assert.True(t, exists, "the layout is stored as a file named after it")
		client.GET(layoutPath(request.Name)).AssertOK().AssertJSON(saved)
		client.GET("/api/print/layouts").AssertJSONCount("", len(defaultLayoutNames)+1)
	})

	t.Run("replaces the layout", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()
		layout := newLayoutSaveRequest(h.Faker)
		client.PUT(layoutPath(layout.Name), layout).AssertOK()
		replaced := newLayoutSaveRequest(h.Faker, func(r *api.LayoutSaveRequest) { r.Name = layout.Name })

		saved := client.PUT(layoutPath(layout.Name), replaced).AssertOK().Decode[api.LayoutDetailResponse]()

		assert.Equal(t, replaced.Description, saved.Description)
		client.GET(layoutPath(layout.Name)).AssertOK().AssertJSON(saved)
		client.GET("/api/print/layouts").AssertJSONCount("", len(defaultLayoutNames)+1)
	})

	t.Run("stores the longest name of wide characters", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()
		// Characters of 4 bytes each give the longest file name the name limit allows
		name := make([]rune, 60)
		for i := range name {
			name[i] = rune(h.Faker.IntRange(0x1F600, 0x1F64F))
		}
		layout := newLayoutSaveRequest(h.Faker, func(r *api.LayoutSaveRequest) { r.Name = string(name) })

		client.PUT(layoutPath(layout.Name), layout).AssertOK()

		client.GET(layoutPath(layout.Name)).AssertOK().AssertJSONPath("name", layout.Name)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			change func(*api.LayoutSaveRequest, *gofakeit.Faker)
			field  string
			tag    string
		}{
			{
				name:   "missing name",
				change: func(r *api.LayoutSaveRequest, _ *gofakeit.Faker) { r.Name = "" },
				field:  "name",
				tag:    "required",
			},
			{
				name:   "empty description",
				change: func(r *api.LayoutSaveRequest, _ *gofakeit.Faker) { empty := ""; r.Description = &empty },
				field:  "description",
				tag:    "min",
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
				name:   "name too long",
				change: func(r *api.LayoutSaveRequest, _ *gofakeit.Faker) { r.Name = strings.Repeat(r.Name[:1], 61) },
				field:  "name",
				tag:    "max",
			},
			{
				name:   "blank name",
				change: func(r *api.LayoutSaveRequest, _ *gofakeit.Faker) { r.Name = " " },
				field:  "name",
				tag:    "invalid",
			},
			{
				name:   "name outside the folder",
				change: func(r *api.LayoutSaveRequest, _ *gofakeit.Faker) { r.Name = ".." },
				field:  "name",
				tag:    "invalid",
			},
			{
				name:   "name a file system refuses",
				change: func(r *api.LayoutSaveRequest, _ *gofakeit.Faker) { r.Name += ":" },
				field:  "name",
				tag:    "invalid",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := newHarness(t)
				client := h.asAdmin()
				request := newLayoutSaveRequest(h.Faker)
				name := request.Name
				tt.change(&request, h.Faker)
				// The path carries the changed name, so the name check of the store sees it.
				// An empty name has no route and keeps the original one.
				if request.Name != "" {
					name = request.Name
				}

				response := client.PUT(layoutPath(name), request)

				assertFieldError(t, response, tt.field, tt.tag)
				client.GET("/api/print/layouts").AssertJSONCount("", len(defaultLayoutNames))
			})
		}
	})

	t.Run("rejects a name other than the path", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()
		layout := newLayoutSaveRequest(h.Faker)
		pathName := layoutName(h.Faker)

		response := client.PUT(layoutPath(pathName), layout)

		assertFieldError(t, response, "name", "invalid")
		client.GET(layoutPath(pathName)).AssertNotFound()
		client.GET(layoutPath(layout.Name)).AssertNotFound()
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
				request := newLayoutSaveRequest(h.Faker, func(r *api.LayoutSaveRequest) {
					r.Pages[0].Slots[0] = tt.slot
				})

				response := client.PUT(layoutPath(request.Name), request)

				response.AssertBadRequest()
				client.GET(layoutPath(request.Name)).AssertNotFound()
			})
		}
	})

	t.Run("stores nothing for a guest", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		request := newLayoutSaveRequest(h.Faker)

		h.Client.PUT(layoutPath(request.Name), request).AssertUnauthorized()

		h.asAdmin().GET(layoutPath(request.Name)).AssertNotFound()
	})
}

func TestLayoutDeleteHandler(t *testing.T) {
	t.Parallel()

	t.Run("removes the layout", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()

		response := client.DELETE(layoutPath(defaultLayoutNames[0]))

		response.AssertNoContent()
		client.GET(layoutPath(defaultLayoutNames[0])).AssertNotFound()
		client.GET("/api/print/layouts").AssertJSONCount("", len(defaultLayoutNames)-1)
	})

	t.Run("returns not found for an unknown layout", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().DELETE(layoutPath(layoutName(h.Faker)))

		response.AssertNotFound().AssertJSONPath("code", "not_found")
	})

	t.Run("returns not found for a malformed name", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		client := h.asAdmin()

		response := client.DELETE(layoutPath(".."))

		response.AssertNotFound().AssertJSONPath("code", "not_found")
		client.GET("/api/print/layouts").AssertJSONCount("", len(defaultLayoutNames))
	})

	t.Run("keeps the configured layout", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withPrinting("managed"))
		client := h.asAdmin()

		response := client.DELETE(layoutPath(h.Config.Print.Layout))

		response.AssertStatus(http.StatusConflict).AssertJSONPath("code", "conflict")
		client.GET(layoutPath(h.Config.Print.Layout)).AssertOK()
	})

	t.Run("keeps the layout for a guest", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		h.Client.DELETE(layoutPath(defaultLayoutNames[0])).AssertUnauthorized()

		h.asAdmin().GET(layoutPath(defaultLayoutNames[0])).AssertOK()
	})
}

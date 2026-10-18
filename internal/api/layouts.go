package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/barthjs/photopi/internal/domain"
)

// layoutListHandler lists every stored print layout.
//
//	@Summary		List layouts
//	@Description	Lists every stored print layout available to compose a run's captures onto (Admin only).
//	@Tags			print
//	@Produce		json
//	@Success		200	{array}		LayoutResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/print/layouts [get]
func (a *API) layoutListHandler(w http.ResponseWriter, r *http.Request) {
	layouts, err := a.layoutService.List(r.Context())
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	response := make([]LayoutResponse, 0, len(layouts))
	for _, l := range layouts {
		response = append(response, toLayoutResponse(l))
	}

	a.writeJSON(w, r, http.StatusOK, response, nil)
}

// LayoutResponse is the summary of a print layout in the layout list.
type LayoutResponse struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	PaperWidthMM  float64 `json:"paperWidthMM"`
	PaperHeightMM float64 `json:"paperHeightMM"`
	// Slots is the total slot count summed across every page.
	Slots int `json:"slots"`
	Pages int `json:"pages"`
}

// toLayoutResponse maps a layout to its summary response.
func toLayoutResponse(l domain.Layout) LayoutResponse {
	slots := 0
	for _, page := range l.Pages {
		slots += len(page.Slots)
	}

	return LayoutResponse{
		ID:            l.ID,
		Name:          l.Name,
		PaperWidthMM:  l.Paper.WidthMM,
		PaperHeightMM: l.Paper.HeightMM,
		Slots:         slots,
		Pages:         len(l.Pages),
	}
}

// layoutGetHandler returns one stored print layout with its pages and slots.
//
//	@Summary		Get layout
//	@Description	Returns one stored print layout with its pages and slots (Admin only).
//	@Tags			print
//	@Produce		json
//	@Param			id	path		string	true	"Layout ID"
//	@Success		200	{object}	LayoutDetailResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/print/layouts/{id} [get]
func (a *API) layoutGetHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	layout, err := a.layoutService.Get(r.Context(), id)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusOK, toLayoutDetailResponse(*layout), nil)
}

// LayoutDetailResponse is a print layout with its pages and slots.
type LayoutDetailResponse struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Paper    LayoutPaperResponse    `json:"paper"`
	Pages    []LayoutPageResponse   `json:"pages"`
	Branding LayoutBrandingResponse `json:"branding"`
}

// LayoutPageResponse is the slots of one printed page.
type LayoutPageResponse struct {
	Slots []LayoutSlotResponse `json:"slots"`
}

// LayoutSlotResponse places one capture on the page, relative to the page size (0..1).
type LayoutSlotResponse struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
	// Locked only keeps the slot in place in the editor, printing ignores it.
	Locked bool `json:"locked"`
}

// LayoutPaperResponse is the paper size the layout is designed for.
type LayoutPaperResponse struct {
	WidthMM  float64 `json:"widthMM"`
	HeightMM float64 `json:"heightMM"`
}

// LayoutBrandingResponse is the optional caption on the page.
type LayoutBrandingResponse struct {
	Enabled     bool    `json:"enabled"`
	Template    string  `json:"template"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	RotationDeg float64 `json:"rotationDeg"`
	FontSizePt  float64 `json:"fontSizePt,omitzero"`
	Color       string  `json:"color,omitzero"`
}

// LayoutSaveRequest holds the print layout to create or replace.
type LayoutSaveRequest struct {
	Name     string                `json:"name" validate:"required,max=100"`
	Paper    LayoutPaperRequest    `json:"paper"`
	Pages    []LayoutPageRequest   `json:"pages" validate:"required,min=1,max=10,dive"`
	Branding LayoutBrandingRequest `json:"branding"`
}

// LayoutPageRequest is one physical page's worth of slots.
type LayoutPageRequest struct {
	Slots []LayoutSlotRequest `json:"slots" validate:"required,min=1,max=12,dive"`
}

// LayoutSlotRequest is the position and size of one capture on a page, as fractions of the page.
type LayoutSlotRequest struct {
	X float64 `json:"x" validate:"min=0,max=1"`
	Y float64 `json:"y" validate:"min=0,max=1"`
	W float64 `json:"w" validate:"gt=0,max=1"`
	H float64 `json:"h" validate:"gt=0,max=1"`
	// Locked only keeps the slot in place in the editor, printing ignores it.
	Locked bool `json:"locked"`
}

// LayoutPaperRequest is the paper size of a layout.
type LayoutPaperRequest struct {
	// WidthMM/HeightMM are capped at 500mm: RenderCollage allocates a canvas of mm/25.4*renderDPI pixels per side,
	// so an unbounded value risks running out of memory on a device with little RAM.
	// 500mm covers any realistic print size.
	WidthMM  float64 `json:"widthMM" validate:"gt=0,max=500"`
	HeightMM float64 `json:"heightMM" validate:"gt=0,max=500"`
}

// LayoutBrandingRequest is the caption a layout draws onto its pages.
type LayoutBrandingRequest struct {
	Enabled  bool   `json:"enabled"`
	Template string `json:"template" validate:"max=1024"`
	// X/Y are the center of the caption on the page (0..1, same convention as LayoutSlotRequest).
	X           float64 `json:"x" validate:"gte=0,lte=1"`
	Y           float64 `json:"y" validate:"gte=0,lte=1"`
	RotationDeg float64 `json:"rotationDeg" validate:"gte=-180,lte=180"`
	// FontSizePt and Color are optional. Left unset, the caption is drawn with domain.DefaultBrandingFontSizePt in black.
	FontSizePt float64 `json:"fontSizePt" validate:"omitempty,gte=6,lte=72"`
	Color      string  `json:"color" validate:"omitempty,rgbcolor"`
}

// layoutSaveHandler creates or replaces the print layout stored under id.
//
//	@Summary		Save layout
//	@Description	Validates and stores the submitted layout under id, creating it if it doesn't exist yet (Admin only).
//	@Tags			print
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Layout ID"
//	@Param			request	body		LayoutSaveRequest	true	"Layout"
//	@Success		200		{object}	LayoutDetailResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/print/layouts/{id} [put]
func (a *API) layoutSaveHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	req, ok := a.decodeJSON[LayoutSaveRequest](w, r)
	if !ok {
		return
	}

	if err := validateLayoutPages(req.Pages); err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	layout := req.toLayout(id)
	if err := a.layoutService.Save(r.Context(), id, layout); err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusOK, toLayoutDetailResponse(layout), nil)
}

// toLayout maps the request to the layout stored under id.
func (r LayoutSaveRequest) toLayout(id string) domain.Layout {
	pages := make([]domain.LayoutPage, 0, len(r.Pages))
	for _, p := range r.Pages {
		slots := make([]domain.LayoutSlot, 0, len(p.Slots))
		for _, s := range p.Slots {
			slots = append(slots, domain.LayoutSlot{
				X:      s.X,
				Y:      s.Y,
				W:      s.W,
				H:      s.H,
				Locked: s.Locked,
			})
		}
		pages = append(pages, domain.LayoutPage{
			Slots: slots,
		})
	}

	return domain.Layout{
		ID:   id,
		Name: r.Name,
		Paper: domain.LayoutPaper{
			WidthMM:  r.Paper.WidthMM,
			HeightMM: r.Paper.HeightMM,
		},
		Pages: pages,
		Branding: domain.LayoutBranding{
			Enabled:     r.Branding.Enabled,
			Template:    r.Branding.Template,
			X:           r.Branding.X,
			Y:           r.Branding.Y,
			RotationDeg: r.Branding.RotationDeg,
			FontSizePt:  r.Branding.FontSizePt,
			Color:       r.Branding.Color,
		},
	}
}

// toLayoutDetailResponse maps a layout to its detail response.
func toLayoutDetailResponse(l domain.Layout) LayoutDetailResponse {
	pages := make([]LayoutPageResponse, 0, len(l.Pages))
	for _, page := range l.Pages {
		slots := make([]LayoutSlotResponse, 0, len(page.Slots))
		for _, slot := range page.Slots {
			slots = append(slots, LayoutSlotResponse{
				X:      slot.X,
				Y:      slot.Y,
				W:      slot.W,
				H:      slot.H,
				Locked: slot.Locked,
			})
		}
		pages = append(pages, LayoutPageResponse{
			Slots: slots,
		})
	}

	return LayoutDetailResponse{
		ID:   l.ID,
		Name: l.Name,
		Paper: LayoutPaperResponse{
			WidthMM:  l.Paper.WidthMM,
			HeightMM: l.Paper.HeightMM,
		},
		Pages: pages,
		Branding: LayoutBrandingResponse{
			Enabled:     l.Branding.Enabled,
			Template:    l.Branding.Template,
			X:           l.Branding.X,
			Y:           l.Branding.Y,
			RotationDeg: l.Branding.RotationDeg,
			FontSizePt:  l.Branding.FontSizePt,
			Color:       l.Branding.Color,
		},
	}
}

// validateLayoutPages rejects slots that would extend past the page, a check that isn't expressible
// with validator struct tags alone since it compares two fields of the same slot.
func validateLayoutPages(pages []LayoutPageRequest) error {
	for p, page := range pages {
		for i, s := range page.Slots {
			if s.X+s.W > 1 {
				return fmt.Errorf("pages[%d].slots[%d]: x+w must be at most 1, got %g+%g", p, i, s.X, s.W)
			}
			if s.Y+s.H > 1 {
				return fmt.Errorf("pages[%d].slots[%d]: y+h must be at most 1, got %g+%g", p, i, s.Y, s.H)
			}
		}
	}
	return nil
}

// layoutDeleteHandler removes a stored print layout by id.
// The layout the running or the saved configuration prints with cannot be deleted,
// so printing and sharing never lose their layout.
//
//	@Summary		Delete layout
//	@Description	Removes a stored print layout by id. The layout configured for printing cannot be deleted (Admin only).
//	@Tags			print
//	@Param			id	path	string	true	"Layout ID"
//	@Success		204
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/print/layouts/{id} [delete]
func (a *API) layoutDeleteHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if id == a.configService.PrintLayoutID() || id == a.printService.LayoutID() {
		a.conflictResponse(w, r, "the layout is configured for printing")
		return
	}

	if err := a.layoutService.Delete(r.Context(), id); err != nil {
		a.respondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

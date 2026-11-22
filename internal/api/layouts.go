package api

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/go-playground/validator/v10"

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
		response = append(response, toLayoutResponse(l, a.layoutInUse(l.Name)))
	}

	a.writeJSON(w, r, http.StatusOK, response, nil)
}

type LayoutResponse struct {
	Name          string  `json:"name"`
	Description   *string `json:"description"`
	PaperWidthMM  float64 `json:"paperWidthMM"`
	PaperHeightMM float64 `json:"paperHeightMM"`
	// Slots is the total slot count summed across every page.
	Slots int `json:"slots"`
	Pages int `json:"pages"`
	// InUse is true for the layout the running or the saved configuration prints with,
	// which can be neither deleted nor renamed.
	InUse bool `json:"inUse"`
}

// toLayoutResponse maps a layout to its summary response.
func toLayoutResponse(l domain.Layout, inUse bool) LayoutResponse {
	slots := 0
	for _, page := range l.Pages {
		slots += len(page.Slots)
	}

	return LayoutResponse{
		Name:          l.Name,
		Description:   l.Description,
		PaperWidthMM:  l.Paper.WidthMM,
		PaperHeightMM: l.Paper.HeightMM,
		Slots:         slots,
		Pages:         len(l.Pages),
		InUse:         inUse,
	}
}

// layoutGetHandler returns one stored print layout with its pages and slots.
//
//	@Summary		Get layout
//	@Description	Returns one stored print layout with its pages and slots (Admin only).
//	@Tags			print
//	@Produce		json
//	@Param			name	path		string	true	"Layout name"
//	@Success		200		{object}	LayoutDetailResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/print/layouts/{name} [get]
func (a *API) layoutGetHandler(w http.ResponseWriter, r *http.Request) {
	layout, err := a.layoutService.Get(r.Context(), pathParam(r, "name"))
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusOK, toLayoutDetailResponse(*layout), nil)
}

type LayoutDetailResponse struct {
	Name        string                 `json:"name"`
	Description *string                `json:"description"`
	Paper       LayoutPaperResponse    `json:"paper"`
	Pages       []LayoutPageResponse   `json:"pages"`
	Branding    LayoutBrandingResponse `json:"branding"`
}

type LayoutPageResponse struct {
	Slots []LayoutSlotResponse `json:"slots"`
}

// LayoutSlotResponse places one capture on the page relative to the page size.
type LayoutSlotResponse struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
	// Locked only keeps the slot in place in the editor, printing ignores it.
	Locked bool `json:"locked"`
}

type LayoutPaperResponse struct {
	WidthMM  float64 `json:"widthMM"`
	HeightMM float64 `json:"heightMM"`
}

type LayoutBrandingResponse struct {
	Enabled     bool    `json:"enabled"`
	Template    string  `json:"template"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	RotationDeg float64 `json:"rotationDeg"`
	FontSizePt  float64 `json:"fontSizePt,omitzero"`
	Color       string  `json:"color,omitzero"`
}

type LayoutSaveRequest struct {
	// Name matches the name in the path and names the file, so 60 characters of up to 4 bytes
	// stay within the 255 bytes a file name may have.
	Name        string                `json:"name" validate:"required,min=1,max=60"`
	Description *string               `json:"description" validate:"omitempty,min=1,max=1000"`
	Paper       LayoutPaperRequest    `json:"paper"`
	Pages       []LayoutPageRequest   `json:"pages" validate:"required,min=1,max=10,dive"`
	Branding    LayoutBrandingRequest `json:"branding"`
}

type LayoutPageRequest struct {
	Slots []LayoutSlotRequest `json:"slots" validate:"required,min=1,max=12,dive"`
}

// LayoutSlotRequest places one capture on the page relative to the page size.
type LayoutSlotRequest struct {
	X float64 `json:"x" validate:"min=0,max=1"`
	Y float64 `json:"y" validate:"min=0,max=1"`
	W float64 `json:"w" validate:"gt=0,max=1"`
	H float64 `json:"h" validate:"gt=0,max=1"`
	// Locked only keeps the slot in place in the editor, printing ignores it.
	Locked bool `json:"locked"`
}

type LayoutPaperRequest struct {
	// WidthMM is capped, since the collage allocates a canvas of mm/25.4*renderDPI pixels per side,
	// which could run out of memory on a device with little RAM.
	WidthMM float64 `json:"widthMM" validate:"gt=0,max=500"`
	// HeightMM is capped like WidthMM.
	HeightMM float64 `json:"heightMM" validate:"gt=0,max=500"`
}

type LayoutBrandingRequest struct {
	Enabled  bool   `json:"enabled"`
	Template string `json:"template" validate:"max=1024"`
	// X is the horizontal center of the caption relative to the page width.
	X float64 `json:"x" validate:"gte=0,lte=1"`
	// Y is the vertical center of the caption relative to the page height.
	Y           float64 `json:"y" validate:"gte=0,lte=1"`
	RotationDeg float64 `json:"rotationDeg" validate:"gte=-180,lte=180"`
	// FontSizePt is domain.DefaultBrandingFontSizePt when unset.
	FontSizePt float64 `json:"fontSizePt" validate:"omitempty,gte=6,lte=72"`
	// Color is black when unset.
	Color string `json:"color" validate:"omitempty,rgbcolor"`
}

// layoutSaveHandler creates or replaces a print layout.
//
//	@Summary		Save layout
//	@Description	Creates or replaces the print layout with the given name, which has to match the name in the body (Admin only).
//	@Tags			print
//	@Accept			json
//	@Produce		json
//	@Param			name	path		string				true	"Layout name"
//	@Param			request	body		LayoutSaveRequest	true	"Layout"
//	@Success		200		{object}	LayoutDetailResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/print/layouts/{name} [put]
func (a *API) layoutSaveHandler(w http.ResponseWriter, r *http.Request) {
	req, ok := a.decodeJSON[LayoutSaveRequest](w, r)
	if !ok {
		return
	}

	if err := validateLayoutPages(req.Pages); err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	layout := req.toLayout()
	if err := a.layoutService.Save(r.Context(), pathParam(r, "name"), layout); err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeJSON(w, r, http.StatusOK, toLayoutDetailResponse(layout), nil)
}

// toLayout maps the request to the layout it stores.
func (r LayoutSaveRequest) toLayout() domain.Layout {
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
		Name:        r.Name,
		Description: r.Description,
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
		Name:        l.Name,
		Description: l.Description,
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

// validateLayoutPages rejects slots that would extend past the page.
func validateLayoutPages(pages []LayoutPageRequest) error {
	// Struct tags cannot express this rule, since it compares two fields of the same slot
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

// layoutDeleteHandler deletes a print layout, except the one the running or the saved configuration uses.
//
//	@Summary		Delete layout
//	@Description	Deletes the print layout with the given name. The layout configured for printing cannot be deleted (Admin only).
//	@Tags			print
//	@Param			name	path	string	true	"Layout name"
//	@Success		204
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/print/layouts/{name} [delete]
func (a *API) layoutDeleteHandler(w http.ResponseWriter, r *http.Request) {
	name := pathParam(r, "name")

	if a.layoutInUse(name) {
		a.conflictResponse(w, r, "the layout is configured for printing")
		return
	}

	if err := a.layoutService.Delete(r.Context(), name); err != nil {
		a.respondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// layoutInUse reports whether the running or the saved configuration prints with the layout.
func (a *API) layoutInUse(name string) bool {
	return name == a.configService.PrintLayout() || name == a.printService.Layout()
}

// init registers the rules of the layout DTOs with the validator.
func init() {
	// The built-in hexcolor tag also accepts short and alpha forms, which the collage renderer cannot draw,
	// so a layout color is checked for #rrggbb only.
	if err := v.RegisterValidation("rgbcolor", validateRGBColor); err != nil {
		panic(err)
	}
}

// rgbColorPattern matches a color as #rrggbb.
var rgbColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// validateRGBColor reports whether the field holds a color as #rrggbb.
func validateRGBColor(fl validator.FieldLevel) bool {
	return rgbColorPattern.MatchString(fl.Field().String())
}

package domain

// DefaultBrandingFontSizePt is the font size of a caption whose layout sets none.
const DefaultBrandingFontSizePt = 14

// LayoutSlot places one capture of a run on the page as a rectangle relative to the page size.
// Locked only keeps the slot in place in the editor and has no effect on printing.
type LayoutSlot struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	W      float64 `json:"w"`
	H      float64 `json:"h"`
	Locked bool    `json:"locked"`
}

// LayoutPage holds the slots of one printed sheet. The captures of a run fill the pages of a Layout in order,
// so a run with more captures than one page holds continues on the next page.
type LayoutPage struct {
	Slots []LayoutSlot `json:"slots"`
}

// LayoutPaper is the physical page size a Layout is designed for.
type LayoutPaper struct {
	WidthMM  float64 `json:"widthMM"`
	HeightMM float64 `json:"heightMM"`
}

// LayoutBranding optionally renders the event name and date as a caption on the page,
// placed by its center relative to the page size like a LayoutSlot.
type LayoutBranding struct {
	Enabled     bool    `json:"enabled"`
	Template    string  `json:"template"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	RotationDeg float64 `json:"rotationDeg"`
	FontSizePt  float64 `json:"fontSizePt,omitzero"`
	Color       string  `json:"color,omitempty"`
}

// Layout describes how the captures of a run are composed onto one or more pages.
// It carries JSON tags because the layout store persists it as JSON.
type Layout struct {
	Name        string         `json:"name"`
	Description *string        `json:"description"`
	Paper       LayoutPaper    `json:"paper"`
	Pages       []LayoutPage   `json:"pages"`
	Branding    LayoutBranding `json:"branding"`
}

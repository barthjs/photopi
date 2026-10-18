package domain

// Collage is one rendered page of the collage of a run with everything its file name is made of.
type Collage struct {
	// Event is the slug of the event folder the collage is filed under.
	Event string
	// Prefix starts the file name, like the captures of the run.
	Prefix string
	Run    int
	// Page is the index of the page in the layout, starting at zero.
	Page int
	// LayoutHash ends the file name, so a changed layout renders new pages.
	LayoutHash string
}

package domain

// RunListFilter defines filtering options for listing runs. A nil Event lists the runs of every event.
type RunListFilter struct {
	// Event is the slug of the event folder the runs lie in.
	Event  *string
	Limit  int
	Offset int
}

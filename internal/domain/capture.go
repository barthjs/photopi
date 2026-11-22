package domain

import (
	"time"
)

// Capture is a new photo of a run with everything its file name is made of.
type Capture struct {
	// Event is the slug of the event folder the capture is filed under.
	Event string
	// Prefix starts the file name.
	Prefix     string
	Run        int
	ImageIndex int
	// TakenAt ends the file name. It only informs, since the clock of a booth without internet may be wrong.
	TakenAt time.Time
}

// CaptureFile is a stored capture of a run.
type CaptureFile struct {
	Path string
	// Event is the slug of the event folder the file lies in.
	Event      string
	Run        int
	ImageIndex int
	// TakenAt is read from the file name in the local time zone, zero if it cannot be parsed.
	TakenAt time.Time
}

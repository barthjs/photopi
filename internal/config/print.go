package config

import (
	"strings"

	"github.com/barthjs/photopi/internal/config/structconf"
)

const (
	// PrintSubmissionModeManaged sends a PDF through the CUPS filters and the driver of the printer,
	// which scale it to the paper of the layout. This is the default and works with any working driver.
	PrintSubmissionModeManaged = "managed"
	// PrintSubmissionModeRaw sends the PDF untouched (lp -o raw), so its page size alone determines the output.
	// It is meant for a printer with its own PDF interpreter.
	PrintSubmissionModeRaw = "raw"
)

// PrintConfig holds the print settings.
type PrintConfig struct {
	Enabled        bool   `mapstructure:"enabled" json:"enabled" default:"false" usage:"Print the composed layout of every run through a CUPS queue"`
	PrinterName    string `mapstructure:"printer_name" json:"printerName" usage:"CUPS queue name as listed by lpstat -p, required when printing is enabled"`
	LayoutID       string `mapstructure:"layout_id" json:"layoutId" default:"strip-3" usage:"Layout file without .json that composes the print and share collages"`
	SubmissionMode string `mapstructure:"submission_mode" json:"submissionMode" default:"managed" usage:"How print jobs reach CUPS, managed lets the printer driver convert and scale the PDF, raw sends it untouched to a printer with its own PDF interpreter"`
}

// validatePrint normalizes and checks the print settings. They only apply when printing is enabled.
func validatePrint(cfg *PrintConfig) error {
	var p structconf.Problems

	if !cfg.Enabled {
		return p.Err()
	}

	if cfg.PrinterName == "" {
		p.Addf("print.printer_name is required when print.enabled is true")
	}

	cfg.LayoutID = strings.TrimSpace(cfg.LayoutID)
	if cfg.LayoutID == "" {
		p.Addf("print.layout_id is required when print.enabled is true")
	}
	if len(cfg.LayoutID) > maxShortText {
		p.Addf("print.layout_id must be at most %d characters", maxShortText)
	}

	switch cfg.SubmissionMode {
	case PrintSubmissionModeRaw, PrintSubmissionModeManaged:
	default:
		p.Addf("print.submission_mode must be %q or %q, got %q", PrintSubmissionModeRaw, PrintSubmissionModeManaged, cfg.SubmissionMode)
	}

	return p.Err()
}

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePrint(t *testing.T) {
	t.Parallel()

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()

		assertRejects(t, []rejection{
			{
				name: "without printer",
				change: func(c *Config) {
					c.Print.Enabled = true
					c.Print.PrinterName = ""
				},
				wantErr: "print.printer_name",
			},
			{
				name: "without layout",
				change: func(c *Config) {
					c.Print.Enabled = true
					c.Print.PrinterName = "p"
					c.Print.Layout = " "
				},
				wantErr: "print.layout",
			},
			{
				name: "unknown submission mode",
				change: func(c *Config) {
					c.Print.Enabled = true
					c.Print.PrinterName = "p"
					c.Print.SubmissionMode = "fax"
				},
				wantErr: "print.submission_mode",
			},
		})
	})

	t.Run("ignores disabled printing", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t)
		cfg.Print.SubmissionMode = "fax"

		err := Validate(&cfg)

		assert.NoError(t, err)
	})
}

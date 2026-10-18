package camera

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReportsProblems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		stderr string
		want   bool
	}{
		{
			name:   "reports a warning",
			stderr: "[0:00:01] [12]  WARN RPiSdn sdn.cpp:40 Using legacy SDN tuning",
			want:   true,
		},
		{
			name:   "reports an error",
			stderr: "*** Error: could not claim the USB device ***",
			want:   true,
		},
		{
			name:   "ignores progress output",
			stderr: "Still capture image received",
			want:   false,
		},
		{
			name:   "ignores empty output",
			stderr: "",
			want:   false,
		},
		{
			name: "ignores the hint of every immediate capture",
			stderr: "WARNING: Capture will not make use of temporal denoise\n" +
				"         Consider using the --zsl option for best results, for example:\n" +
				"         rpicam-still --zsl -o -\nStill capture image received",
			want: false,
		},
		{
			name: "reports a problem after the hint",
			stderr: "WARNING: Capture will not make use of temporal denoise\n" +
				"         Consider using the --zsl option\nERROR: failed to start camera",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := reportsProblems(tt.stderr)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTailBuffer_String(t *testing.T) {
	t.Parallel()

	t.Run("keeps the last complete lines", func(t *testing.T) {
		t.Parallel()

		buffer := newTailBuffer(16)
		_, _ = io.WriteString(buffer, "first line\n")
		_, _ = io.WriteString(buffer, "second\nthird\n")

		got := buffer.String()

		assert.Equal(t, "second\nthird", got, "the output starts at the first complete line")
	})
}

package camera_test

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/camera"
)

func TestExposureTime_Duration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		time camera.ExposureTime
		want time.Duration
	}{
		{
			name: "returns a fraction of a second",
			time: "1/125",
			want: 8 * time.Millisecond,
		},
		{
			name: "returns a fraction without an exact number of microseconds",
			time: "1/60",
			want: time.Second / 60,
		},
		{
			name: "returns zero for the automatic exposure",
			time: camera.ExposureTimeAuto,
			want: 0,
		},
		{
			name: "returns zero without a fraction",
			time: "fast",
			want: 0,
		},
		{
			name: "returns zero for a zero denominator",
			time: "1/0",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.time.Duration()

			assert.Equal(t, tt.want, got)
		})
	}
}

// TestNew is not parallel, since a subtest replaces PATH.
func TestNew(t *testing.T) {
	t.Run("rejects an unknown source", func(t *testing.T) {
		t.Parallel()

		cam, err := camera.New(zerolog.Nop(), camera.Options{
			Source: "pinhole",
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, `unknown source "pinhole"`)
		assert.Nil(t, cam)
	})

	t.Run("rejects a source without its tools", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())

		tests := []struct {
			source camera.Source
			tool   string
		}{
			{
				source: camera.SourceRPiCam,
				tool:   "rpicam-vid",
			},
			{
				source: camera.SourceGphoto,
				tool:   "gphoto2",
			},
		}

		for _, tt := range tests {
			t.Run(string(tt.source), func(t *testing.T) {
				_, err := camera.New(zerolog.Nop(), camera.Options{
					Source: tt.source,
				})

				require.Error(t, err)
				assert.ErrorContains(t, err, `required tool "`+tt.tool+`" not found`)
			})
		}
	})
}

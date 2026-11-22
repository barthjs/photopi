package service

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/domain"
	"github.com/barthjs/photopi/internal/media"
	"github.com/barthjs/photopi/internal/store"
)

func TestPrintService_ListPrinters(t *testing.T) {
	t.Parallel()

	printer := apitest.Faker(t).LetterN(8)

	tests := []struct {
		name    string
		output  string
		err     error
		want    []PrinterInfo
		wantErr bool
	}{
		{
			name:   "returns the printers",
			output: "printer " + printer + " is idle.  enabled since Mon 12 Oct 2026 09:14:03 CEST\n",
			want: []PrinterInfo{
				{
					Name:   printer,
					Status: PrinterIdle,
				},
			},
		},
		{
			name: "returns no printers when lpstat exits non-zero without printers",
			err: &exec.ExitError{
				Stderr: []byte("lpstat: No destinations added.\n"),
			},
			want: nil,
		},
		{
			name: "fails when the scheduler is not running",
			err: &exec.ExitError{
				Stderr: []byte("lpstat: Scheduler is not running.\n"),
			},
			wantErr: true,
		},
		{
			name:    "fails without lpstat",
			err:     errors.New("executable file not found"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(1), 0)
			f.commands.respond = func(commandCall) ([]byte, error) { return []byte(tt.output), tt.err }

			printers, err := f.svc.ListPrinters(t.Context())

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, printers)
			call := f.commands.Calls()[0]
			assert.Equal(t, []string{"-p"}, call.Args)
			assert.Equal(t, lpstatEnv, call.Env, "lpstat answers in English")
		})
	}
}

func TestPrintService_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns not found for an unknown run", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(1), 0)

		_, found := f.svc.Get(1)

		assert.False(t, found)
	})
}

func TestPrintService_Enabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "reports printing as enabled",
			enabled: true,
		},
		{
			name:    "reports printing as disabled",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := NewPrintService(zerolog.Nop(), PrintOptions{
				Enabled: tt.enabled,
			}, nil, nil, nil, nil, nil)

			enabled := svc.Enabled()

			assert.Equal(t, tt.enabled, enabled)
		})
	}
}

func TestPrintService_TestConnection(t *testing.T) {
	t.Parallel()

	printer := apitest.Faker(t).LetterN(8)

	tests := []struct {
		name    string
		printer string
		output  string
		err     error
		wantOK  bool
		message string
	}{
		{
			name:    "reports an idle printer",
			printer: printer,
			output:  lpstatIdle(printer),
			wantOK:  true,
			message: "is idle",
		},
		{
			name:    "reports a disabled printer",
			printer: printer,
			output:  "printer " + printer + " disabled since Mon 12 Oct 2026 09:20:11 CEST -\n\treason unknown\n",
			wantOK:  false,
			message: "is disabled",
		},
		{
			name:    "reports a printer holding new jobs",
			printer: printer,
			output:  "printer " + printer + " is holding new jobs.  enabled since Mon 12 Oct 2026 09:21:00 CEST\n",
			wantOK:  false,
			message: "is holding new jobs",
		},
		{
			name:    "reports a missing printer for unexpected output",
			printer: printer,
			output:  "something else\n",
			wantOK:  false,
			message: "not found",
		},
		{
			name:    "reports the output of a failing lpstat",
			printer: printer,
			output:  "lpstat: Invalid destination name in list \"" + printer + "\".\n",
			err:     errors.New("exit status 1"),
			wantOK:  false,
			message: "Invalid destination name",
		},
		{
			name:    "rejects a missing printer selection",
			printer: "",
			wantOK:  false,
			message: "no printer selected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(1), 0)
			f.commands.respond = func(commandCall) ([]byte, error) { return []byte(tt.output), tt.err }

			ok, message := f.svc.TestConnection(t.Context(), tt.printer)

			assert.Equal(t, tt.wantOK, ok)
			assert.Contains(t, message, tt.message)
		})
	}
}

// squareLayout has one page of 1 x 1 inch with a single slot covering it.
func squareLayout() domain.Layout {
	return domain.Layout{
		Name: "square",
		Paper: domain.LayoutPaper{
			WidthMM:  25.4,
			HeightMM: 25.4,
		},
		Pages: []domain.LayoutPage{
			{
				Slots: []domain.LayoutSlot{
					{
						X: 0,
						Y: 0,
						W: 1,
						H: 1,
					},
				},
			},
		},
	}
}

// fillRect fills rect of img with c.
func fillRect(img *image.RGBA, rect image.Rectangle, c color.Color) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.Set(x, y, c)
		}
	}
}

// testJPEGOf encodes img as JPEG.
func testJPEGOf(t *testing.T, img image.Image) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{
		Quality: 95,
	}))
	return buf.Bytes()
}

func TestPrintService_Begin(t *testing.T) {
	t.Parallel()

	t.Run("marks the run as pending", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(1), 0)

		started := f.svc.Begin(1)

		assert.True(t, started)
		state, found := f.svc.Get(1)
		require.True(t, found)
		assert.Equal(t, JobPending, state.Status)
	})

	t.Run("starts another run on its own", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(1), 0)
		require.True(t, f.svc.Begin(1))

		started := f.svc.Begin(2)

		assert.True(t, started, "another run prints on its own")
	})

	t.Run("rejects a second start while the print is in flight", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(1), 0)
		require.True(t, f.svc.Begin(1))

		started := f.svc.Begin(1)

		assert.False(t, started, "a print in flight is not started twice")
	})
}

func TestPrintService_Print(t *testing.T) {
	t.Parallel()

	printer := apitest.Faker(t).LetterN(8)

	t.Run("sends every page as one PDF job filling the paper in managed mode", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{
			Enabled:        true,
			PrinterName:    printer,
			SubmissionMode: PrintSubmissionManaged,
		}, twoSlotLayout(2), 3)
		f.commands.respond = cupsAnswers(lpstatIdle(printer), "")

		err := f.svc.Print(t.Context(), 1)

		require.NoError(t, err)
		calls := lpCalls(f.commands.Calls())
		require.Len(t, calls, 1)
		assert.Equal(t, []string{"-d", printer, "-o", "media=Custom.2.00x1.00in", "-o", "print-scaling=fill"}, calls[0].Args[:6])
		assert.True(t, bytes.HasPrefix(calls[0].File, []byte("%PDF-1.4")))
		assert.Contains(t, string(calls[0].File), "/Count 2", "both pages are in one document")
		assertTempFileRemoved(t, calls[0])
		state, found := f.svc.Get(1)
		require.True(t, found)
		assert.Equal(t, JobState{
			Status: JobDone,
		}, state)
	})

	t.Run("sends the PDF untouched in raw mode", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{
			Enabled:        true,
			PrinterName:    printer,
			SubmissionMode: PrintSubmissionRaw,
		}, twoSlotLayout(1), 2)
		f.commands.respond = cupsAnswers(lpstatIdle(printer), "")

		err := f.svc.Print(t.Context(), 1)

		require.NoError(t, err)
		calls := lpCalls(f.commands.Calls())
		require.Len(t, calls, 1)
		assert.Equal(t, []string{"-d", printer, "-o", "raw"}, calls[0].Args[:4])
		assert.Contains(t, string(calls[0].File), "/MediaBox [0 0 144.00 72.00]", "2 x 1 inch in points")
		assertTempFileRemoved(t, calls[0])
	})

	t.Run("rejects printing while it is off", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{
			Enabled:     false,
			PrinterName: printer,
		}, twoSlotLayout(1), 1)

		err := f.svc.Print(t.Context(), 1)

		assert.ErrorContains(t, err, "not enabled")
		assert.Empty(t, f.commands.Calls())
	})

	t.Run("rejects printing without a printer", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{
			Enabled: true,
		}, twoSlotLayout(1), 1)

		err := f.svc.Print(t.Context(), 1)

		assert.ErrorContains(t, err, "no printer configured")
		assert.Empty(t, f.commands.Calls())
	})

	t.Run("returns not found without captures", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{
			Enabled:     true,
			PrinterName: printer,
		}, twoSlotLayout(1), 0)
		f.commands.respond = cupsAnswers(lpstatIdle(printer), "")

		err := f.svc.Print(t.Context(), 1)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("rejects an unavailable queue without printing", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{
			Enabled:     true,
			PrinterName: printer,
		}, twoSlotLayout(1), 2)
		f.commands.respond = cupsAnswers(lpstatIdle(printer), queuedJobs(printer, maxPendingPrintJobs+1))

		err := f.svc.Print(t.Context(), 1)

		assert.ErrorIs(t, err, ErrPrintQueueUnavailable)
		assert.Empty(t, lpCalls(f.commands.Calls()))
		state, found := f.svc.Get(1)
		require.True(t, found)
		assert.Equal(t, JobState{
			Status: JobFailed,
			Reason: JobReasonQueueUnavailable,
		}, state)
	})

	t.Run("reports the output of a failing lp", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{
			Enabled:     true,
			PrinterName: printer,
		}, twoSlotLayout(1), 2)
		answers := cupsAnswers(lpstatIdle(printer), "")
		f.commands.respond = func(call commandCall) ([]byte, error) {
			if call.Name == "lp" {
				return []byte("lp: Destination \"" + printer + "\" is not accepting jobs.\n"),
					errors.New("exit status 1")
			}
			return answers(call)
		}

		err := f.svc.Print(t.Context(), 1)

		assert.ErrorContains(t, err, "is not accepting jobs")
		state, _ := f.svc.Get(1)
		assert.Equal(t, JobState{
			Status: JobFailed,
			Reason: JobReasonFailed,
		}, state)
	})
}

func TestPrintService_RenderCollage(t *testing.T) {
	t.Parallel()

	t.Run("places the captures in the slots at print resolution", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(1), 2)

		paths, err := f.svc.RenderCollage(t.Context(), 1)

		require.NoError(t, err)
		require.Len(t, paths, 1)
		assert.Equal(t, "prints/summer-party", path.Dir(paths[0]), "collages are stored per event")
		assert.Regexp(t, `^x_000001-collage-1_[0-9a-f]{8}\.jpg$`, path.Base(paths[0]))
		img := decodeJPEG(t, readFile(t, f.storage, paths[0]))
		assert.Equal(t, image.Rect(0, 0, 600, 300), img.Bounds(), "2 x 1 inch at 300 dpi")
		left, right := pixel(img, 150, 150), pixel(img, 450, 150)
		assert.Greater(t, left.R, uint8(200), "the first capture fills the left slot")
		assert.Greater(t, right.G, uint8(200), "the second capture fills the right slot")
	})

	t.Run("continues on the next page and leaves missing slots blank", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(2), 3)

		paths, err := f.svc.RenderCollage(t.Context(), 1)

		require.NoError(t, err)
		require.Len(t, paths, 2)
		assert.Contains(t, paths[1], "-collage-2_")
		second := decodeJPEG(t, readFile(t, f.storage, paths[1]))
		assert.Greater(t, pixel(second, 150, 150).B, uint8(200), "the third capture starts the second page")
		blank := pixel(second, 450, 150)
		assert.Greater(t, blank.R, uint8(240))
		assert.Greater(t, blank.G, uint8(240), "an unused slot stays white")
	})

	t.Run("leaves out pages no capture reaches", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(3), 2)

		paths, err := f.svc.RenderCollage(t.Context(), 1)

		require.NoError(t, err)
		assert.Len(t, paths, 1)
		assert.Len(t, listFiles(t, f.storage, "prints/"), 1)
	})

	t.Run("crops a photo from the center to the slot", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, squareLayout(), 0)
		// A wide photo with blue edges and a green center the size of a square
		photo := solidImage(120, 40, color.RGBA{
			B: 255,
			A: 255,
		})
		fillRect(photo, image.Rect(40, 0, 80, 40), color.RGBA{
			G: 255,
			A: 255,
		})
		putCapture(t, f.storage, 1, 1, testJPEGOf(t, photo))

		paths, err := f.svc.RenderCollage(t.Context(), 1)

		require.NoError(t, err)
		img := decodeJPEG(t, readFile(t, f.storage, paths[0]))
		for _, x := range []int{30, 150, 270} {
			assert.Greater(t, pixel(img, x, 150).G, uint8(200), "only the center of the photo fills the slot at x %d", x)
		}
	})

	t.Run("turns a photo upright by its EXIF orientation", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, squareLayout(), 0)
		// Stored with red on the left and blue on the right, shown turned clockwise, so red is on top
		photo := solidImage(40, 40, color.RGBA{
			B: 255,
			A: 255,
		})
		fillRect(photo, image.Rect(0, 0, 20, 40), color.RGBA{
			R: 255,
			A: 255,
		})
		putCapture(t, f.storage, 1, 1, withOrientation(t, testJPEGOf(t, photo), 6))

		paths, err := f.svc.RenderCollage(t.Context(), 1)

		require.NoError(t, err)
		img := decodeJPEG(t, readFile(t, f.storage, paths[0]))
		assert.Greater(t, pixel(img, 150, 40).R, uint8(200), "the left edge of the stored photo is on top")
		assert.Greater(t, pixel(img, 150, 260).B, uint8(200), "the right edge of the stored photo is at the bottom")
	})

	t.Run("draws the branding caption", func(t *testing.T) {
		t.Parallel()

		layout := twoSlotLayout(1)
		layout.Pages[0].Slots = []domain.LayoutSlot{
			{
				X: 0,
				Y: 0,
				W: 0.05,
				H: 0.1,
			},
		}
		layout.Branding = domain.LayoutBranding{
			Enabled:  true,
			Template: "{{eventName}}",
			X:        0.5,
			Y:        0.5,
			Color:    "#000000",
		}
		f := newPrintFixture(t, PrintOptions{}, layout, 1)

		paths, err := f.svc.RenderCollage(t.Context(), 1)

		require.NoError(t, err)
		img := decodeJPEG(t, readFile(t, f.storage, paths[0]))
		darkest := uint8(255)
		for x := 200; x < 400; x++ {
			if r := pixel(img, x, 150).R; r < darkest {
				darkest = r
			}
		}
		assert.Less(t, darkest, uint8(100), "the caption text is drawn across the center")
		corner := pixel(img, 590, 290)
		assert.Greater(t, corner.R, uint8(240), "the rest of the page stays white")
	})

	t.Run("draws the branding caption in black for an invalid color", func(t *testing.T) {
		t.Parallel()

		layout := twoSlotLayout(1)
		layout.Pages[0].Slots = []domain.LayoutSlot{
			{
				X: 0,
				Y: 0,
				W: 0.05,
				H: 0.1,
			},
		}
		layout.Branding = domain.LayoutBranding{
			Enabled:  true,
			Template: "{{eventName}}",
			X:        0.5,
			Y:        0.5,
			Color:    "not a color",
		}
		f := newPrintFixture(t, PrintOptions{}, layout, 1)

		paths, err := f.svc.RenderCollage(t.Context(), 1)

		require.NoError(t, err)
		img := decodeJPEG(t, readFile(t, f.storage, paths[0]))
		darkest := pixel(img, 200, 150)
		for x := 200; x < 400; x++ {
			if current := pixel(img, x, 150); current.R < darkest.R {
				darkest = current
			}
		}
		assert.Less(t, darkest.R, uint8(100), "the caption falls back to black")
		assert.Less(t, darkest.G, uint8(100), "the caption falls back to black")
		assert.Less(t, darkest.B, uint8(100), "the caption falls back to black")
	})

	t.Run("reuses rendered pages", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(1), 2)
		first, err := f.svc.RenderCollage(t.Context(), 1)
		require.NoError(t, err)
		putFile(t, f.storage, first[0], []byte("cached"))

		paths, err := f.svc.RenderCollage(t.Context(), 1)

		require.NoError(t, err)
		assert.Equal(t, first, paths)
		assert.Equal(t, []byte("cached"), readFile(t, f.storage, paths[0]), "the cached page is returned unchanged")
	})

	t.Run("renders every page again when one is missing", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(2), 4)
		paths, err := f.svc.RenderCollage(t.Context(), 1)
		require.NoError(t, err)
		require.NoError(t, f.storage.Delete(t.Context(), paths[1]))
		putFile(t, f.storage, paths[0], []byte("stale"))

		_, err = f.svc.RenderCollage(t.Context(), 1)

		require.NoError(t, err)
		decodeJPEG(t, readFile(t, f.storage, paths[0]))
		decodeJPEG(t, readFile(t, f.storage, paths[1]))
	})

	t.Run("renders new pages for a changed layout and removes the old ones", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(1), 2)
		old, err := f.svc.RenderCollage(t.Context(), 1)
		require.NoError(t, err)
		changed := twoSlotLayout(1)
		changed.Pages[0].Slots[1].W = 0.4
		err = f.layouts.Save(t.Context(), changed.Name, changed)
		require.NoError(t, err)

		paths, err := f.svc.RenderCollage(t.Context(), 1)

		require.NoError(t, err)
		assert.NotEqual(t, old, paths, "the file name carries the hash of the layout")
		assert.Equal(t, paths, listFiles(t, f.storage, "prints/"), "the collage of the old layout is removed")
	})

	t.Run("keeps the collages of other runs", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(1), 2)
		other := "prints/summer-party/x_000002-collage-1_00000000.jpg"
		putFile(t, f.storage, other, []byte("other run"))

		_, err := f.svc.RenderCollage(t.Context(), 1)

		require.NoError(t, err)
		assert.Equal(t, []byte("other run"), readFile(t, f.storage, other))
	})

	t.Run("renders once for concurrent callers", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(2), 4)
		const callers = 4
		results := make([][]string, callers)
		var wg sync.WaitGroup

		for i := range callers {
			wg.Go(func() {
				paths, err := f.svc.RenderCollage(t.Context(), 1)
				assert.NoError(t, err)
				results[i] = paths
			})
		}
		wg.Wait()

		for _, paths := range results {
			assert.Equal(t, results[0], paths)
		}
		assert.Len(t, listFiles(t, f.storage, "prints/"), 2)
	})

	t.Run("rejects a capture that is not a JPEG", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(1), 0)
		putCapture(t, f.storage, 1, 1, []byte("not a jpeg"))

		_, err := f.svc.RenderCollage(t.Context(), 1)

		assert.ErrorContains(t, err, "decoding")
	})

	t.Run("returns not found for an unknown layout", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{
			Layout: "missing",
		}, twoSlotLayout(1), 1)

		_, err := f.svc.RenderCollage(t.Context(), 1)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("returns not found without captures", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(1), 0)

		_, err := f.svc.RenderCollage(t.Context(), 1)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("rejects a layout without slots", func(t *testing.T) {
		t.Parallel()

		layout := twoSlotLayout(1)
		layout.Pages[0].Slots = nil
		f := newPrintFixture(t, PrintOptions{}, layout, 1)

		_, err := f.svc.RenderCollage(t.Context(), 1)

		assert.ErrorContains(t, err, "no slot to fill")
		assert.Empty(t, listFiles(t, f.storage, "prints/"), "no page is rendered")
	})
}

// lpCalls returns the calls of lp.
func lpCalls(calls []commandCall) []commandCall {
	var result []commandCall
	for _, call := range calls {
		if call.Name == "lp" {
			result = append(result, call)
		}
	}
	return result
}

// assertTempFileRemoved checks that the temporary file passed to lp is gone after the call.
func assertTempFileRemoved(t *testing.T, call commandCall) {
	t.Helper()

	_, err := os.Stat(call.Args[len(call.Args)-1])
	assert.ErrorIs(t, err, os.ErrNotExist, "the temporary file is removed")
}

func TestFilledPages(t *testing.T) {
	t.Parallel()

	t.Run("leaves out pages without slots or captures", func(t *testing.T) {
		t.Parallel()

		layout := twoSlotLayout(3)
		layout.Pages[1].Slots = nil

		pages := filledPages(layout, 3)

		require.Len(t, pages, 2, "a page without slots and a page no capture reaches are left out")
		assert.Equal(t, 0, pages[0].index)
		assert.Len(t, pages[0].slots, 2)
		assert.Equal(t, 2, pages[1].index)
		assert.Len(t, pages[1].slots, 1, "only the slots with a capture are filled")
		assert.Equal(t, 2, pages[1].firstCapture)
	})
}

func TestJPEGOrientation(t *testing.T) {
	t.Parallel()

	t.Run("returns upright for a JPEG without EXIF", func(t *testing.T) {
		t.Parallel()

		plain := testJPEG(t, 8, 8, color.White)

		orientation := jpegOrientation(plain)

		assert.Equal(t, 1, orientation, "a JPEG without EXIF is upright")
	})

	t.Run("returns the stored orientation", func(t *testing.T) {
		t.Parallel()

		plain := testJPEG(t, 8, 8, color.White)

		for orientation := range uint16(8) {
			got := jpegOrientation(withOrientation(t, plain, orientation+1))

			assert.Equal(t, int(orientation+1), got)
		}
	})

	t.Run("returns upright for an unknown orientation", func(t *testing.T) {
		t.Parallel()

		plain := testJPEG(t, 8, 8, color.White)

		orientation := jpegOrientation(withOrientation(t, plain, 9))

		assert.Equal(t, 1, orientation, "an unknown orientation counts as upright")
	})

	t.Run("returns upright for data that is not a JPEG", func(t *testing.T) {
		t.Parallel()

		orientation := jpegOrientation([]byte("not a jpeg"))

		assert.Equal(t, 1, orientation)
	})

	t.Run("returns upright for a segment without a marker", func(t *testing.T) {
		t.Parallel()

		orientation := jpegOrientation([]byte{0xFF, 0xD8, 0x00, 0xE1, 0x00, 0x02})

		assert.Equal(t, 1, orientation, "data that is no segment ends the search")
	})

	t.Run("ignores a truncated segment", func(t *testing.T) {
		t.Parallel()

		plain := testJPEG(t, 8, 8, color.White)

		orientation := jpegOrientation(withOrientation(t, plain, 6)[:12])

		assert.Equal(t, 1, orientation, "a truncated segment is ignored")
	})
}

func TestExifOrientation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tiff []byte
		want int
	}{
		{
			name: "reads the orientation in little endian order",
			tiff: []byte{
				'I', 'I', 42, 0, 8, 0, 0, 0,
				1, 0,
				0x12, 0x01, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0,
			},
			want: 6,
		},
		{
			name: "reads the orientation in big endian order",
			tiff: []byte{
				'M', 'M', 0, 42, 0, 0, 0, 8,
				0, 1,
				0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, 3, 0, 0,
			},
			want: 3,
		},
		{
			name: "skips other tags before the orientation",
			tiff: []byte{
				'I', 'I', 42, 0, 8, 0, 0, 0,
				2, 0,
				0x0F, 0x01, 2, 0, 1, 0, 0, 0, 0, 0, 0, 0,
				0x12, 0x01, 3, 0, 1, 0, 0, 0, 8, 0, 0, 0,
			},
			want: 8,
		},
		{
			name: "returns upright for data shorter than a header",
			tiff: []byte{'I', 'I', 42, 0, 8, 0, 0},
			want: 1,
		},
		{
			name: "returns upright for an unknown byte order",
			tiff: []byte{
				'X', 'X', 42, 0, 8, 0, 0, 0,
				1, 0,
				0x12, 0x01, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0,
			},
			want: 1,
		},
		{
			name: "returns upright for a wrong magic number",
			tiff: []byte{
				'I', 'I', 43, 0, 8, 0, 0, 0,
				1, 0,
				0x12, 0x01, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0,
			},
			want: 1,
		},
		{
			name: "returns upright for a directory inside the header",
			tiff: []byte{
				'I', 'I', 42, 0, 4, 0, 0, 0,
				1, 0,
				0x12, 0x01, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0,
			},
			want: 1,
		},
		{
			name: "returns upright for a directory beyond the data",
			tiff: []byte{'I', 'I', 42, 0, 64, 0, 0, 0},
			want: 1,
		},
		{
			name: "returns upright for a truncated directory entry",
			tiff: []byte{
				'I', 'I', 42, 0, 8, 0, 0, 0,
				1, 0,
				0x12, 0x01, 3, 0, 1, 0,
			},
			want: 1,
		},
		{
			name: "returns upright for a directory without the orientation",
			tiff: []byte{
				'I', 'I', 42, 0, 8, 0, 0, 0,
				1, 0,
				0x0F, 0x01, 2, 0, 1, 0, 0, 0, 0, 0, 0, 0,
			},
			want: 1,
		},
		{
			name: "returns upright for an orientation of zero",
			tiff: []byte{
				'I', 'I', 42, 0, 8, 0, 0, 0,
				1, 0,
				0x12, 0x01, 3, 0, 1, 0, 0, 0, 0, 0, 0, 0,
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orientation := exifOrientation(tt.tiff)

			assert.Equal(t, tt.want, orientation)
		})
	}
}

// withOrientation inserts an EXIF APP1 segment with the orientation tag right after the start of a JPEG.
func withOrientation(t *testing.T, data []byte, orientation uint16) []byte {
	t.Helper()

	require.True(t, bytes.HasPrefix(data, []byte{0xFF, 0xD8}))

	// TIFF header in big-endian order, one image directory with the orientation tag
	var tiff bytes.Buffer
	_, _ = tiff.WriteString("MM")
	_ = binary.Write(&tiff, binary.BigEndian, uint16(42))
	_ = binary.Write(&tiff, binary.BigEndian, uint32(8))
	_ = binary.Write(&tiff, binary.BigEndian, uint16(1))
	_ = binary.Write(&tiff, binary.BigEndian, uint16(0x0112))
	_ = binary.Write(&tiff, binary.BigEndian, uint16(3))
	_ = binary.Write(&tiff, binary.BigEndian, uint32(1))
	_ = binary.Write(&tiff, binary.BigEndian, orientation)
	_ = binary.Write(&tiff, binary.BigEndian, uint16(0))
	_ = binary.Write(&tiff, binary.BigEndian, uint32(0))

	payload := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	segment := []byte{0xFF, 0xE1}
	segment = binary.BigEndian.AppendUint16(segment, uint16(len(payload)+2))
	segment = append(segment, payload...)

	result := append([]byte{0xFF, 0xD8}, segment...)
	return append(result, data[2:]...)
}

func TestCenterCrop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		bounds image.Rectangle
		aspect float64
		want   image.Rectangle
	}{
		{
			name:   "crops the sides of a wide image",
			bounds: image.Rect(0, 0, 120, 40),
			aspect: 1,
			want:   image.Rect(40, 0, 80, 40),
		},
		{
			name:   "crops top and bottom of a tall image",
			bounds: image.Rect(0, 0, 40, 120),
			aspect: 1,
			want:   image.Rect(0, 40, 40, 80),
		},
		{
			name:   "keeps a matching image whole",
			bounds: image.Rect(0, 0, 60, 40),
			aspect: 1.5,
			want:   image.Rect(0, 0, 60, 40),
		},
		{
			name:   "keeps empty bounds",
			bounds: image.Rect(0, 0, 0, 40),
			aspect: 1,
			want:   image.Rect(0, 0, 0, 40),
		},
		{
			name:   "keeps the bounds for an aspect ratio that is not positive",
			bounds: image.Rect(0, 0, 120, 40),
			aspect: 0,
			want:   image.Rect(0, 0, 120, 40),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			crop := centerCrop(tt.bounds, tt.aspect)

			assert.Equal(t, tt.want, crop)
		})
	}
}

func TestOrientRGBA(t *testing.T) {
	t.Parallel()

	// A 2 x 1 image, a on the left and b on the right
	a := color.RGBA{
		R: 255,
		A: 255,
	}
	b := color.RGBA{
		B: 255,
		A: 255,
	}
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	src.Set(0, 0, a)
	src.Set(1, 0, b)

	tests := []struct {
		name        string
		orientation int
		want        []color.RGBA
		size        image.Point
	}{
		{
			name:        "keeps the image for orientation 1",
			orientation: 1,
			want:        []color.RGBA{a, b},
			size:        image.Pt(2, 1),
		},
		{
			name:        "mirrors the image horizontally for orientation 2",
			orientation: 2,
			want:        []color.RGBA{b, a},
			size:        image.Pt(2, 1),
		},
		{
			name:        "turns the image half around for orientation 3",
			orientation: 3,
			want:        []color.RGBA{b, a},
			size:        image.Pt(2, 1),
		},
		{
			name:        "mirrors the image vertically for orientation 4",
			orientation: 4,
			want:        []color.RGBA{a, b},
			size:        image.Pt(2, 1),
		},
		{
			name:        "transposes the image for orientation 5",
			orientation: 5,
			want:        []color.RGBA{a, b},
			size:        image.Pt(1, 2),
		},
		{
			name:        "turns the image clockwise for orientation 6",
			orientation: 6,
			want:        []color.RGBA{a, b},
			size:        image.Pt(1, 2),
		},
		{
			name:        "transverses the image for orientation 7",
			orientation: 7,
			want:        []color.RGBA{b, a},
			size:        image.Pt(1, 2),
		},
		{
			name:        "turns the image counterclockwise for orientation 8",
			orientation: 8,
			want:        []color.RGBA{b, a},
			size:        image.Pt(1, 2),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dst := orientRGBA(src, tt.orientation)

			require.Equal(t, tt.size, dst.Bounds().Size())
			var got []color.RGBA
			for y := range tt.size.Y {
				for x := range tt.size.X {
					got = append(got, dst.RGBAAt(x, y))
				}
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPrintService_CheckQueue(t *testing.T) {
	t.Parallel()

	printer := apitest.Faker(t).LetterN(8)

	accepted := []struct {
		name   string
		status string
		jobs   string
	}{
		{
			name:   "accepts an idle queue",
			status: lpstatIdle(printer),
			jobs:   queuedJobs(printer, maxPendingPrintJobs),
		},
		{
			name:   "accepts a printing queue",
			status: "printer " + printer + " now printing " + printer + "-1.  enabled since Mon 12 Oct 2026 09:15:44 CEST\n",
		},
	}

	for _, tt := range accepted {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newPrintFixture(t, PrintOptions{
				PrinterName: printer,
			}, twoSlotLayout(1), 0)
			f.commands.respond = cupsAnswers(tt.status, tt.jobs)

			err := f.svc.CheckQueue(t.Context())

			assert.NoError(t, err)
		})
	}

	t.Run("checks the queue of an instance", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{
			PrinterName: printer + "/glossy",
		}, twoSlotLayout(1), 0)
		f.commands.respond = cupsAnswers(lpstatIdle(printer), "")

		err := f.svc.CheckQueue(t.Context())

		require.NoError(t, err)
		for _, call := range f.commands.Calls() {
			assert.Equal(t, printer, call.Args[1])
		}
	})

	rejected := []struct {
		name   string
		status string
		jobs   string
	}{
		{
			name:   "rejects a disabled queue",
			status: "printer " + printer + " disabled since Mon 12 Oct 2026 09:20:11 CEST -\n\tPaused\n",
		},
		{
			name:   "rejects a queue holding new jobs",
			status: "printer " + printer + " is holding new jobs.  enabled since Mon 12 Oct 2026 09:21:00 CEST\n",
		},
		{
			name:   "rejects a missing queue",
			status: lpstatIdle("other"),
		},
		{
			name:   "rejects a queue with too many waiting jobs",
			status: lpstatIdle(printer),
			jobs:   queuedJobs(printer, maxPendingPrintJobs+1),
		},
	}

	for _, tt := range rejected {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newPrintFixture(t, PrintOptions{
				PrinterName: printer,
			}, twoSlotLayout(1), 0)
			f.commands.respond = cupsAnswers(tt.status, tt.jobs)

			err := f.svc.CheckQueue(t.Context())

			assert.ErrorIs(t, err, ErrPrintQueueUnavailable)
		})
	}

	t.Run("rejects the queue when lpstat fails", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{
			PrinterName: printer,
		}, twoSlotLayout(1), 0)
		f.commands.respond = func(commandCall) ([]byte, error) {
			return nil, &exec.ExitError{
				Stderr: []byte("lpstat: Scheduler is not running.\n"),
			}
		}

		err := f.svc.CheckQueue(t.Context())

		assert.ErrorIs(t, err, ErrPrintQueueUnavailable)
	})
}

func TestParsePrinterStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output string
		want   []PrinterInfo
	}{
		{
			name: "returns every queue state lpstat reports",
			output: "printer DNP_DS_RX1 is idle.  enabled since Mon 12 Oct 2026 09:14:03 CEST\n" +
				"printer Office now printing Office-12.  enabled since Mon 12 Oct 2026 09:15:44 CEST\n" +
				"printer Canon disabled since Mon 12 Oct 2026 09:20:11 CEST -\n" +
				"\tPaused\n" +
				"printer Selphy is holding new jobs.  enabled since Mon 12 Oct 2026 09:21:00 CEST\n",
			want: []PrinterInfo{
				{
					Name:   "DNP_DS_RX1",
					Status: PrinterIdle,
				},
				{
					Name:   "Office",
					Status: PrinterPrinting,
				},
				{
					Name:   "Canon",
					Status: PrinterDisabled,
				},
				{
					Name:   "Selphy",
					Status: PrinterHolding,
				},
			},
		},
		{
			name:   "returns the instance of a queue",
			output: "printer DNP_DS_RX1/glossy is idle.  enabled since Mon 12 Oct 2026 09:14:03 CEST\n",
			want: []PrinterInfo{
				{
					Name:   "DNP_DS_RX1/glossy",
					Status: PrinterIdle,
				},
			},
		},
		{
			name:   "ignores the reason of a disabled queue",
			output: "printer Canon disabled since Mon 12 Oct 2026 09:20:11 CEST -\n\treason unknown\n",
			want: []PrinterInfo{
				{
					Name:   "Canon",
					Status: PrinterDisabled,
				},
			},
		},
		{
			name:   "returns no printers without configured ones",
			output: "lpstat: No destinations added.\n",
			want:   nil,
		},
		{
			name:   "returns no printers for empty output",
			output: "",
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			printers := parsePrinterStatus([]byte(tt.output))

			assert.Equal(t, tt.want, printers)
		})
	}
}

func TestCountQueuedJobs(t *testing.T) {
	t.Parallel()

	t.Run("counts the jobs of the queue", func(t *testing.T) {
		t.Parallel()

		output := queuedJobs("DNP", 2) + queuedJobs("DNP_other", 1)

		count := countQueuedJobs([]byte(output), "DNP")

		assert.Equal(t, 2, count, "jobs of a queue whose name starts alike do not count")
	})

	t.Run("returns zero without output", func(t *testing.T) {
		t.Parallel()

		count := countQueuedJobs(nil, "DNP")

		assert.Zero(t, count)
	})
}

// queuedJobs is how lpstat -o lists count jobs of a queue.
func queuedJobs(name string, count int) string {
	var b strings.Builder
	for i := range count {
		_, _ = b.WriteString(name + "-" + string(rune('1'+i)) + "          photopi        1024   Mon 12 Oct 2026 09:30:00 CEST\n")
	}
	return b.String()
}

func TestParseJobID(t *testing.T) {
	t.Parallel()

	t.Run("returns the job ID", func(t *testing.T) {
		t.Parallel()

		id := parseJobID([]byte("request id is DNP_DS_RX1-42 (1 file(s))\n"))

		assert.Equal(t, "DNP_DS_RX1-42", id)
	})

	t.Run("returns nothing for empty output", func(t *testing.T) {
		t.Parallel()

		id := parseJobID([]byte(""))

		assert.Empty(t, id)
	})
}

func TestWriteTempFile(t *testing.T) {
	t.Parallel()

	t.Run("stores the content", func(t *testing.T) {
		t.Parallel()

		content := []byte(apitest.Faker(t).LoremIpsumSentence(6))

		p, err := writeTempFile("photopi-test-*.bin", content)

		require.NoError(t, err)
		t.Cleanup(func() { _ = os.Remove(p) })
		data, err := os.ReadFile(p)
		require.NoError(t, err)
		assert.Equal(t, content, data)
		assert.True(t, strings.HasSuffix(p, ".bin"))
	})
}

func TestWrapJPEGsAsPDF(t *testing.T) {
	t.Parallel()

	t.Run("returns a PDF with a page per image", func(t *testing.T) {
		t.Parallel()

		pages := [][]byte{
			testJPEG(t, 30, 20, color.White),
			testJPEG(t, 30, 20, color.Black),
		}

		pdf, err := wrapJPEGsAsPDF(pages, 101.6, 152.4)

		require.NoError(t, err)
		text := string(pdf)
		assert.True(t, strings.HasPrefix(text, "%PDF-1.4\n"))
		assert.True(t, strings.HasSuffix(text, "%%EOF"))
		assert.Contains(t, text, "/Kids [4 0 R 6 0 R] /Count 2")
		assert.Equal(t, 2, strings.Count(text, "/MediaBox [0 0 288.00 432.00]"), "4 x 6 inch in points on every page")
		assert.Equal(t, 2, strings.Count(text, "/Width 30 /Height 20"))
		assert.Contains(t, text, "xref\n0 8\n", "catalog, page tree, content, and a page and an image per page")
	})

	t.Run("rejects data that is not a JPEG", func(t *testing.T) {
		t.Parallel()

		_, err := wrapJPEGsAsPDF([][]byte{[]byte("not a jpeg")}, 10, 10)

		assert.Error(t, err)
	})

	t.Run("rejects a document without pages", func(t *testing.T) {
		t.Parallel()

		_, err := wrapJPEGsAsPDF(nil, 10, 10)

		assert.Error(t, err)
	})
}

func TestPrintService_Resume(t *testing.T) {
	t.Parallel()

	printer := apitest.Faker(t).LetterN(8)

	t.Run("runs the CUPS command for a known queue", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(1), 0)
		f.commands.respond = cupsAnswers(lpstatIdle(printer), "")

		err := f.svc.Resume(t.Context(), printer)

		require.NoError(t, err)
		calls := f.commands.Calls()
		require.Len(t, calls, 2)
		assert.Equal(t, "cupsenable", calls[1].Name)
		assert.Equal(t, []string{"--release", printer}, calls[1].Args)
	})

	t.Run("returns not found for an unknown queue", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(1), 0)
		f.commands.respond = cupsAnswers(lpstatIdle(printer), "")

		err := f.svc.Resume(t.Context(), "-a")

		assert.ErrorIs(t, err, apperr.ErrNotFound)
		assert.Len(t, f.commands.Calls(), 1, "only lpstat runs")
	})

	t.Run("reports a failing command", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(1), 0)
		f.commands.respond = failingCUPSCommand(printer, "cupsenable")

		err := f.svc.Resume(t.Context(), printer)

		assert.ErrorContains(t, err, "Forbidden")
	})
}

func TestPrintService_Cancel(t *testing.T) {
	t.Parallel()

	printer := apitest.Faker(t).LetterN(8)

	t.Run("runs the CUPS command for a known queue", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(1), 0)
		f.commands.respond = cupsAnswers(lpstatIdle(printer), "")

		err := f.svc.Cancel(t.Context(), printer)

		require.NoError(t, err)
		calls := f.commands.Calls()
		require.Len(t, calls, 2)
		assert.Equal(t, "cancel", calls[1].Name)
		assert.Equal(t, []string{"-a", printer}, calls[1].Args)
	})

	t.Run("returns not found for an unknown queue", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(1), 0)
		f.commands.respond = cupsAnswers(lpstatIdle(printer), "")

		err := f.svc.Cancel(t.Context(), "-a")

		assert.ErrorIs(t, err, apperr.ErrNotFound)
		assert.Len(t, f.commands.Calls(), 1, "only lpstat runs")
	})

	t.Run("reports a failing command", func(t *testing.T) {
		t.Parallel()

		f := newPrintFixture(t, PrintOptions{}, twoSlotLayout(1), 0)
		f.commands.respond = failingCUPSCommand(printer, "cancel")

		err := f.svc.Cancel(t.Context(), printer)

		assert.ErrorContains(t, err, "Forbidden")
	})
}

// failingCUPSCommand answers like cupsAnswers for an idle printer, but lets command fail.
func failingCUPSCommand(printer, command string) func(commandCall) ([]byte, error) {
	answers := cupsAnswers(lpstatIdle(printer), "")
	return func(call commandCall) ([]byte, error) {
		if call.Name == command {
			return []byte("Forbidden"), errors.New("exit status 1")
		}
		return answers(call)
	}
}

// printFixture is a print service with a fake CUPS, a stored layout, and a run with captures.
type printFixture struct {
	svc      *PrintService
	storage  media.Storage
	layouts  *store.LayoutStore
	commands *fakeCommands
}

// fixtureEventName is the event of the print fixture, its collages are stored below its slug.
const fixtureEventName = "Summer Party"

// newPrintFixture builds a print service with the layout stored and a run with the given number of colored captures.
func newPrintFixture(t *testing.T, cfg PrintOptions, layout domain.Layout, captures int) printFixture {
	t.Helper()

	storage := newTestStorage(t)
	commands := &fakeCommands{}
	layouts := store.NewLayoutStore(storage)
	require.NoError(t, layouts.Save(t.Context(), layout.Name, layout))

	colors := []color.Color{
		color.RGBA{
			R: 255,
			A: 255,
		},
		color.RGBA{
			G: 255,
			A: 255,
		},
		color.RGBA{
			B: 255,
			A: 255,
		},
	}
	for i := range captures {
		putCapture(t, storage, 1, i+1, testJPEG(t, 40, 40, colors[i%len(colors)]))
	}

	if cfg.Layout == "" {
		cfg.Layout = layout.Name
	}
	cfg.EventName = fixtureEventName
	cfg.FilePrefix = "x"
	runs, captureStore := newTestStores(storage)
	return printFixture{
		svc: NewPrintService(
			zerolog.Nop(),
			cfg,
			commands,
			runs,
			captureStore,
			store.NewCollageStore(storage),
			layouts,
		),
		storage:  storage,
		layouts:  layouts,
		commands: commands,
	}
}

// putCapture stores data as a capture of run in the folder of the fixture event.
func putCapture(t *testing.T, storage media.Storage, run, index int, data []byte) {
	t.Helper()

	putFile(t, storage, fmt.Sprintf("captures/summer-party/x_%06d-%02d_20261014-120000.jpg", run, index), data)
}

// cupsAnswers answers lpstat -p with status, lpstat -o with jobs, and lp with its request ID.
func cupsAnswers(status, jobs string) func(commandCall) ([]byte, error) {
	return func(call commandCall) ([]byte, error) {
		switch {
		case call.Name == "lpstat" && call.Args[0] == "-p":
			return []byte(status), nil
		case call.Name == "lpstat" && call.Args[0] == "-o":
			return []byte(jobs), nil
		case call.Name == "lp":
			return []byte("request id is " + call.Args[1] + "-42 (1 file(s))\n"), nil
		default:
			return nil, nil
		}
	}
}

// lpstatIdle is how lpstat -p reports an idle queue.
func lpstatIdle(name string) string {
	return "printer " + name + " is idle.  enabled since Mon 12 Oct 2026 09:14:03 CEST\n"
}

func TestParseHexColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    color.Color
		wantErr bool
	}{
		{
			name:  "parses a color with hash",
			input: "#ff8000",
			want: color.RGBA{
				R: 255,
				G: 128,
				B: 0,
				A: 255,
			},
		},
		{
			name:  "parses a color without hash",
			input: "0000ff",
			want: color.RGBA{
				B: 255,
				A: 255,
			},
		},
		{
			name:    "rejects a color that is too short",
			input:   "#fff",
			want:    color.Black,
			wantErr: true,
		},
		{
			name:    "rejects a color that is not hex",
			input:   "#zzzzzz",
			want:    color.Black,
			wantErr: true,
		},
		{
			name:    "rejects an empty color",
			input:   "",
			want:    color.Black,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseHexColor(tt.input)

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestMMToPixels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mm   float64
		want int
	}{
		{
			name: "converts one inch at print resolution",
			mm:   25.4,
			want: 300,
		},
		{
			name: "converts four inches at print resolution",
			mm:   101.6,
			want: 1200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pixels := mmToPixels(tt.mm)

			assert.Equal(t, tt.want, pixels)
		})
	}
}

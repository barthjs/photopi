package api_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/api"
)

func TestPrintAccess(t *testing.T) {
	t.Parallel()

	routes := []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/print/printers",
			arrange: func(h *harness) (string, any) {
				idlePrinter(h)
				return "/api/print/printers", nil
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/print/test",
			arrange: func(h *harness) (string, any) {
				body := api.PrintTestRequest{
					PrinterName: idlePrinter(h),
				}
				return "/api/print/test", body
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusOK,
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/print/resume",
			arrange: func(h *harness) (string, any) {
				body := api.PrintResumeRequest{
					PrinterName: idlePrinter(h),
				}
				return "/api/print/resume", body
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/print/cancel",
			arrange: func(h *harness) (string, any) {
				body := api.PrintCancelRequest{
					PrinterName: idlePrinter(h),
				}
				return "/api/print/cancel", body
			},
			statuses: map[*clientKind]int{
				clientGuest: http.StatusUnauthorized,
				clientAdmin: http.StatusNoContent,
			},
		},
	}

	testAccess(t, routes)

	t.Run("runs nothing for a guest", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		for _, route := range routes {
			path, body := route.arrange(h)
			h.Client.Do(route.method, path, body).AssertStatus(route.statuses[clientGuest])
		}

		assert.Zero(t, h.Commands.count())
	})
}

// idlePrinter lets lpstat report a random, idle printer and returns its name.
func idlePrinter(h *harness) string {
	printer := h.Faker.LetterN(8)
	h.Commands.answer("lpstat", lpstatLine(printer, "idle"), nil)
	return printer
}

func TestPrinterListHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the printers", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		idle, disabled := h.Faker.LetterN(8), h.Faker.LetterN(8)
		h.Commands.answer("lpstat", lpstatLine(idle, "idle")+lpstatLine(disabled, "disabled"), nil)

		response := h.asAdmin().GET("/api/print/printers")

		response.AssertOK().AssertJSON([]api.PrinterResponse{
			{
				Name:   idle,
				Status: "idle",
			},
			{
				Name:   disabled,
				Status: "disabled",
			},
		})
		calls := h.Commands.callsOf("lpstat")
		require.Len(t, calls, 1)
		assert.Equal(t, []string{"-p"}, calls[0].Args)
	})

	t.Run("returns no printers without queues", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().GET("/api/print/printers")

		response.AssertOK().AssertJSONCount("", 0)
	})
}

func TestPrintTestHandler(t *testing.T) {
	t.Parallel()

	t.Run("reports whether the printer is ready", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			status string
			err    error
			ok     bool
		}{
			{
				name:   "idle",
				status: "idle",
				ok:     true,
			},
			{
				name:   "printing",
				status: "printing",
				ok:     true,
			},
			{
				name:   "disabled",
				status: "disabled",
				ok:     false,
			},
			{
				name:   "holding new jobs",
				status: "holding",
				ok:     false,
			},
			{
				name: "unknown",
				err:  errFake,
				ok:   false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := newHarness(t)
				printer := h.Faker.LetterN(8)
				output := ""
				if tt.status != "" {
					output = lpstatLine(printer, tt.status)
				}
				h.Commands.answer("lpstat", output, tt.err)

				result := h.asAdmin().POST("/api/print/test", api.PrintTestRequest{
					PrinterName: printer,
				}).AssertOK().Decode[api.ConfigTestResponse]()

				assert.Equal(t, tt.ok, result.OK)
				calls := h.Commands.callsOf("lpstat")
				require.Len(t, calls, 1)
				assert.Equal(t, []string{"-p", printer}, calls[0].Args)
				assert.Empty(t, h.Commands.callsOf("lp"), "testing a printer must not print")
			})
		}
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().POST("/api/print/test", api.PrintTestRequest{})

		assertFieldError(t, response, "printerName", "required")
	})
}

func TestPrintResumeHandler(t *testing.T) {
	t.Parallel()

	testPrintQueueAction(t, printQueueAction{
		path: "/api/print/resume",
		request: func(printer string) any {
			return api.PrintResumeRequest{
				PrinterName: printer,
			}
		},
		command: "cupsenable",
		args: func(printer string) []string {
			return []string{"--release", printer}
		},
	})
}

func TestPrintCancelHandler(t *testing.T) {
	t.Parallel()

	testPrintQueueAction(t, printQueueAction{
		path: "/api/print/cancel",
		request: func(printer string) any {
			return api.PrintCancelRequest{
				PrinterName: printer,
			}
		},
		command: "cancel",
		args: func(printer string) []string {
			return []string{"-a", printer}
		},
	})
}

// printQueueAction is an endpoint that runs one CUPS command on a printer queue.
type printQueueAction struct {
	path    string
	request func(printer string) any
	command string
	args    func(printer string) []string
}

// testPrintQueueAction runs the subtests every endpoint shares that runs a CUPS command on a printer queue.
func testPrintQueueAction(t *testing.T, action printQueueAction) {
	t.Helper()

	t.Run("runs the CUPS command for a known queue", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		printer := h.Faker.LetterN(8)
		h.Commands.answer("lpstat", lpstatLine(printer, "disabled"), nil)

		response := h.asAdmin().POST(action.path, action.request(printer))

		response.AssertNoContent()
		calls := h.Commands.callsOf(action.command)
		require.Len(t, calls, 1)
		assert.Equal(t, action.args(printer), calls[0].Args)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		response := h.asAdmin().POST(action.path, action.request(""))

		assertFieldError(t, response, "printerName", "required")
	})

	t.Run("returns not found for an unknown queue", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.Commands.answer("lpstat", lpstatLine(h.Faker.LetterN(8), "idle"), nil)

		response := h.asAdmin().POST(action.path, action.request(h.Faker.LetterN(9)))

		response.AssertNotFound()
		assert.Empty(t, h.Commands.callsOf(action.command))
	})

	t.Run("reports a failing command", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		printer := h.Faker.LetterN(8)
		h.Commands.answer("lpstat", lpstatLine(printer, "idle"), nil)
		h.Commands.answer(action.command, h.Faker.LoremIpsumSentence(4), errFake)

		response := h.asAdmin().POST(action.path, action.request(printer))

		response.AssertStatus(http.StatusInternalServerError)
	})
}

// lpstatLine is how lpstat -p reports one printer queue in the given status.
func lpstatLine(name, status string) string {
	const since = "Mon 12 Oct 2026 09:14:03 CEST"
	switch status {
	case "printing":
		return fmt.Sprintf("printer %s now printing %s-1.  enabled since %s\n", name, name, since)
	case "disabled":
		return fmt.Sprintf("printer %s disabled since %s -\n\treason unknown\n", name, since)
	case "holding":
		return fmt.Sprintf("printer %s is holding new jobs.  enabled since %s\n", name, since)
	default:
		return fmt.Sprintf("printer %s is idle.  enabled since %s\n", name, since)
	}
}

package api_test

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/api"
	"github.com/barthjs/photopi/internal/config"
)

func TestShareAccess(t *testing.T) {
	t.Parallel()

	// The client of the guest who took the run reaches it, every other client gets a 404
	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/share/{run}/cloud-upload",
			arrange: func(h *harness) (string, any) {
				return startedShareJob(h, "cloud-upload"), nil
			},
			statuses: map[*clientKind]int{
				clientOtherGuest: http.StatusNotFound,
				clientGuest:      http.StatusOK,
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/share/{run}/cloud-upload",
			arrange: func(h *harness) (string, any) {
				return sharedRunPath(h, "cloud-upload"), nil
			},
			statuses: map[*clientKind]int{
				clientOtherGuest: http.StatusNotFound,
				clientGuest:      http.StatusAccepted,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/share/{run}/email",
			arrange: func(h *harness) (string, any) {
				return startedShareJob(h, "email"), nil
			},
			statuses: map[*clientKind]int{
				clientOtherGuest: http.StatusNotFound,
				clientGuest:      http.StatusOK,
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/share/{run}/email",
			arrange: func(h *harness) (string, any) {
				body := api.ShareEmailRequest{
					Email: h.Faker.Email(),
				}
				return sharedRunPath(h, "email"), body
			},
			statuses: map[*clientKind]int{
				clientOtherGuest: http.StatusNotFound,
				clientGuest:      http.StatusAccepted,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/share/{run}/print",
			arrange: func(h *harness) (string, any) {
				return startedShareJob(h, "print"), nil
			},
			statuses: map[*clientKind]int{
				clientOtherGuest: http.StatusNotFound,
				clientGuest:      http.StatusOK,
			},
		},
		{
			method: http.MethodPost,
			path:   "/api/share/{run}/print",
			arrange: func(h *harness) (string, any) {
				return sharedRunPath(h, "print"), nil
			},
			statuses: map[*clientKind]int{
				clientOtherGuest: http.StatusNotFound,
				clientGuest:      http.StatusAccepted,
			},
		},
	}, withEmailSharing(), withCloud(&fakeCloudProvider{}), withPrinting(config.PrintSubmissionModeRaw))

	t.Run("shares nothing for another client", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withEmailSharing())
		run := h.completeRun()

		response := h.newClient().POST(sharePath(run)+"/email", api.ShareEmailRequest{
			Email: h.Faker.Email(),
		})

		response.AssertNotFound()
		assert.Empty(t, h.Mail.sent())
	})
}

// startedShareJob completes a run, lets its guest start the job, waits for it, and returns the path of the job.
func startedShareJob(h *harness, job string) string {
	h.t.Helper()

	jobPath := sharedRunPath(h, job)
	var body any
	if job == "email" {
		body = api.ShareEmailRequest{
			Email: h.Faker.Email(),
		}
	}
	h.Client.POST(jobPath, body).AssertStatus(http.StatusAccepted)
	h.awaitJob(jobPath)
	return jobPath
}

// sharedRunPath completes a run and returns the path of its job.
func sharedRunPath(h *harness, job string) string {
	h.t.Helper()

	return sharePath(h.completeRun()) + "/" + job
}

func TestShareCloudUploadGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns not found for an unknown upload", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withCloud(&fakeCloudProvider{}))
		run := h.completeRun()

		response := h.Client.GET(sharePath(run) + "/cloud-upload")

		response.AssertNotFound()
	})
}

func TestShareCloudUploadHandler(t *testing.T) {
	t.Parallel()

	t.Run("uploads the captures in the background", func(t *testing.T) {
		t.Parallel()
		provider := &fakeCloudProvider{}
		h := newHarness(t, func(h *harness) {
			provider.url = h.Faker.URL()
			h.Deps.CloudProvider = provider
		})
		run := h.completeRun()

		response := h.Client.POST(sharePath(run)+"/cloud-upload", nil)

		response.AssertStatus(http.StatusAccepted)
		assert.Equal(t, api.ShareJobResponse{
			Status: "done",
			URL:    provider.url,
		}, h.awaitJob(sharePath(run)+"/cloud-upload"))
		assert.Equal(t, []int{h.Config.Capture.ImagesPerRun}, provider.uploads())
	})

	t.Run("reports a failing upload", func(t *testing.T) {
		t.Parallel()
		provider := &fakeCloudProvider{
			uploadErr: errFake,
		}
		h := newHarness(t, withCloud(provider))
		run := h.completeRun()

		response := h.Client.POST(sharePath(run)+"/cloud-upload", nil)

		response.AssertStatus(http.StatusAccepted)
		assert.Equal(t, "failed", h.awaitJob(sharePath(run)+"/cloud-upload").Status)
	})

	t.Run("returns not found for an unknown run", func(t *testing.T) {
		t.Parallel()
		provider := &fakeCloudProvider{}
		h := newHarness(t, withCloud(provider))

		response := h.Client.POST(sharePath(h.Faker.IntRange(1, 1000))+"/cloud-upload", nil)

		response.AssertNotFound()
		assert.Empty(t, provider.uploads())
	})

	t.Run("returns not found for a run without captures", func(t *testing.T) {
		t.Parallel()
		provider := &fakeCloudProvider{}
		h := newHarness(t, withCloud(provider))
		run := h.startRun()

		response := h.Client.POST(sharePath(run)+"/cloud-upload", nil)

		response.AssertNotFound()
		assert.Empty(t, provider.uploads())
	})

	t.Run("rejects uploading when cloud sharing is disabled", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()

		response := h.Client.POST(sharePath(run)+"/cloud-upload", nil)

		response.AssertBadRequest()
	})
}

func TestShareEmailGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns not found for an unknown email", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withEmailSharing())
		run := h.completeRun()

		response := h.Client.GET(sharePath(run) + "/email")

		response.AssertNotFound()
	})
}

func TestShareEmailHandler(t *testing.T) {
	t.Parallel()

	t.Run("sends the captures in the background", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withEmailSharing())
		run := h.completeRun()
		recipient := h.Faker.Email()

		response := h.Client.POST(sharePath(run)+"/email", api.ShareEmailRequest{
			Email: recipient,
		})

		response.AssertStatus(http.StatusAccepted)
		assert.Equal(t, api.ShareJobResponse{
			Status: "done",
		}, h.awaitJob(sharePath(run)+"/email"))
		sent := h.Mail.sent()
		require.Len(t, sent, 1)
		assert.Equal(t, recipient, sent[0].To)
		assert.Len(t, sent[0].Attachments, h.Config.Capture.ImagesPerRun)
		assert.NotEmpty(t, sent[0].Subject)
	})

	t.Run("adds the collage when printing", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withEmailSharing(), withPrinting(config.PrintSubmissionModeManaged))
		run := h.completeRun()

		response := h.Client.POST(sharePath(run)+"/email", api.ShareEmailRequest{
			Email: h.Faker.Email(),
		})

		response.AssertStatus(http.StatusAccepted)
		h.awaitJob(sharePath(run) + "/email")
		sent := h.Mail.sent()
		require.Len(t, sent, 1)
		assert.Len(t, sent[0].Attachments, h.Config.Capture.ImagesPerRun+1)
		assert.Empty(t, h.Commands.callsOf("lp"), "sharing must not print")
	})

	t.Run("reports a failing delivery", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withEmailSharing())
		h.Mail.fail(errFake)
		run := h.completeRun()

		response := h.Client.POST(sharePath(run)+"/email", api.ShareEmailRequest{
			Email: h.Faker.Email(),
		})

		response.AssertStatus(http.StatusAccepted)
		assert.Equal(t, api.ShareJobResponse{
			Status: "failed",
			Reason: "failed",
		}, h.awaitJob(sharePath(run)+"/email"))
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withEmailSharing())
		run := h.completeRun()

		response := h.Client.POST(sharePath(run)+"/email", api.ShareEmailRequest{
			Email: h.Faker.Word(),
		})

		assertFieldError(t, response, "email", "email")
		assert.Empty(t, h.Mail.sent())
	})

	t.Run("returns not found for an unknown run", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withEmailSharing())

		response := h.Client.POST(sharePath(h.Faker.IntRange(1, 1000))+"/email", api.ShareEmailRequest{
			Email: h.Faker.Email(),
		})

		response.AssertNotFound()
	})

	t.Run("returns not found for a run without captures", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withEmailSharing())
		run := h.startRun()

		response := h.Client.POST(sharePath(run)+"/email", api.ShareEmailRequest{
			Email: h.Faker.Email(),
		})

		response.AssertNotFound()
		assert.Empty(t, h.Mail.sent())
	})

	t.Run("rejects sharing when email is disabled", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withConfig(func(cfg *config.Config) { cfg.Share.Email.Enabled = false }))
		run := h.completeRun()

		response := h.Client.POST(sharePath(run)+"/email", api.ShareEmailRequest{
			Email: h.Faker.Email(),
		})

		response.AssertBadRequest()
		assert.Empty(t, h.Mail.sent())
	})

	t.Run("rejects an email while one of the run is being sent", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withEmailSharing())
		h.Mail.hold(t)
		run := h.completeRun()
		h.Client.POST(sharePath(run)+"/email", api.ShareEmailRequest{
			Email: h.Faker.Email(),
		}).AssertStatus(http.StatusAccepted)

		response := h.Client.POST(sharePath(run)+"/email", api.ShareEmailRequest{
			Email: h.Faker.Email(),
		})

		response.AssertStatus(http.StatusConflict)
		assert.Equal(t, api.ShareJobResponse{
			Status: "pending",
		}, h.Client.GET(sharePath(run) + "/email").AssertOK().Decode[api.ShareJobResponse]())
	})

	t.Run("is rate limited", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withEmailSharing())
		run := h.completeRun()
		for range 3 {
			h.Client.POST(sharePath(run)+"/email", api.ShareEmailRequest{
				Email: h.Faker.Email(),
			}).AssertStatus(http.StatusAccepted)
			h.awaitJob(sharePath(run) + "/email")
		}

		response := h.Client.POST(sharePath(run)+"/email", api.ShareEmailRequest{
			Email: h.Faker.Email(),
		})

		response.AssertTooManyRequests()
		assert.Len(t, h.Mail.sent(), 3)
	})

	t.Run("is rate limited per run", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withEmailSharing())
		run := h.completeRun()
		for range 3 {
			h.Client.POST(sharePath(run)+"/email", api.ShareEmailRequest{
				Email: h.Faker.Email(),
			}).AssertStatus(http.StatusAccepted)
			h.awaitJob(sharePath(run) + "/email")
		}
		other := h.completeRun()

		response := h.Client.POST(sharePath(other)+"/email", api.ShareEmailRequest{
			Email: h.Faker.Email(),
		})

		response.AssertStatus(http.StatusAccepted)
		h.awaitJob(sharePath(other) + "/email")
		assert.Len(t, h.Mail.sent(), 4)
	})
}

// withEmailSharing lets guests email their captures.
func withEmailSharing() option {
	return withConfig(func(cfg *config.Config) { cfg.Share.Email.Enabled = true })
}

func TestSharePrintGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns not found for an unknown print job", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withPrinting(config.PrintSubmissionModeManaged))
		run := h.completeRun()

		response := h.Client.GET(sharePath(run) + "/print")

		response.AssertNotFound()
	})
}

func TestSharePrintHandler(t *testing.T) {
	t.Parallel()

	t.Run("submits a managed job in the background", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withPrinting(config.PrintSubmissionModeManaged))
		run := h.completeRun()

		response := h.Client.POST(sharePath(run)+"/print", nil)

		response.AssertStatus(http.StatusAccepted)
		assert.Equal(t, api.ShareJobResponse{
			Status: "done",
		}, h.awaitJob(sharePath(run)+"/print"))
		calls := h.Commands.callsOf("lp")
		require.Len(t, calls, 1)
		assert.Equal(t, []string{"-d", h.Config.Print.PrinterName, "-o"}, calls[0].Args[:3])
		assert.True(t, strings.HasPrefix(calls[0].Args[3], "media=Custom."))
		assert.Equal(t, []string{"-o", "print-scaling=fill"}, calls[0].Args[4:6])
		assert.True(t, bytes.HasPrefix(calls[0].File, []byte("%PDF")), "a managed job sends a PDF")
	})

	t.Run("submits a raw job", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withPrinting(config.PrintSubmissionModeRaw))
		run := h.completeRun()

		response := h.Client.POST(sharePath(run)+"/print", nil)

		response.AssertStatus(http.StatusAccepted)
		assert.Equal(t, "done", h.awaitJob(sharePath(run)+"/print").Status)
		calls := h.Commands.callsOf("lp")
		require.Len(t, calls, 1)
		assert.Equal(t, []string{"-d", h.Config.Print.PrinterName, "-o", "raw"}, calls[0].Args[:4])
		assert.True(t, bytes.HasPrefix(calls[0].File, []byte("%PDF")), "a raw job sends a PDF")
	})

	t.Run("reports a failing printer", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withPrinting(config.PrintSubmissionModeManaged))
		h.Commands.answer("lp", h.Faker.LoremIpsumSentence(6), errFake)
		run := h.completeRun()

		response := h.Client.POST(sharePath(run)+"/print", nil)

		response.AssertStatus(http.StatusAccepted)
		assert.Equal(t, api.ShareJobResponse{
			Status: "failed",
			Reason: "failed",
		}, h.awaitJob(sharePath(run)+"/print"))
	})

	t.Run("returns not found for an unknown run", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withPrinting(config.PrintSubmissionModeRaw))

		response := h.Client.POST(sharePath(h.Faker.IntRange(1, 1000))+"/print", nil)

		response.AssertNotFound()
		assert.Empty(t, h.Commands.callsOf("lp"))
	})

	t.Run("returns not found for a run without captures", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withPrinting(config.PrintSubmissionModeRaw))
		run := h.startRun()

		response := h.Client.POST(sharePath(run)+"/print", nil)

		response.AssertNotFound()
		assert.Empty(t, h.Commands.callsOf("lp"))
	})

	t.Run("rejects printing when printing is disabled", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		run := h.completeRun()

		response := h.Client.POST(sharePath(run)+"/print", nil)

		response.AssertBadRequest()
		assert.Empty(t, h.Commands.callsOf("lp"))
	})

	t.Run("rejects an unavailable queue", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			prepare func(h *harness)
		}{
			{
				name: "disabled queue",
				prepare: func(h *harness) {
					h.Commands.answer("lpstat", lpstatLine(h.Config.Print.PrinterName, "disabled"), nil)
				},
			},
			{
				name: "queue holding new jobs",
				prepare: func(h *harness) {
					h.Commands.answer("lpstat", lpstatLine(h.Config.Print.PrinterName, "holding"), nil)
				},
			},
			{
				name: "too many waiting jobs",
				prepare: func(h *harness) {
					h.Commands.answerArg("lpstat", "-o", lpstatJobs(h.Config.Print.PrinterName, 10), nil)
				},
			},
			{
				name: "scheduler not running",
				prepare: func(h *harness) {
					h.Commands.answer("lpstat", "", errFake)
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := newHarness(t, withPrinting(config.PrintSubmissionModeManaged))
				run := h.completeRun()
				tt.prepare(h)

				response := h.Client.POST(sharePath(run)+"/print", nil)

				response.AssertStatus(http.StatusConflict).AssertJSONPath("code", "conflict")
				h.Client.GET(sharePath(run) + "/print").AssertNotFound()
				assert.Empty(t, h.Commands.callsOf("lp"))
			})
		}
	})

	t.Run("is rate limited", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withPrinting(config.PrintSubmissionModeRaw))
		run := h.completeRun()
		for range 3 {
			h.Client.POST(sharePath(run)+"/print", nil).AssertStatus(http.StatusAccepted)
			h.awaitJob(sharePath(run) + "/print")
		}

		response := h.Client.POST(sharePath(run)+"/print", nil)

		response.AssertTooManyRequests()
		assert.Len(t, h.Commands.callsOf("lp"), 3)
	})
}

// awaitJob polls the share job at path until it is no longer pending and returns its final state.
func (h *harness) awaitJob(jobPath string) api.ShareJobResponse {
	h.t.Helper()

	var state api.ShareJobResponse
	require.Eventually(h.t, func() bool {
		state = h.Client.GET(jobPath).AssertOK().Decode[api.ShareJobResponse]()
		return state.Status != "pending"
	}, eventuallyTimeout, eventuallyTick)
	return state
}

// withPrinting turns printing on with a random, idle printer and the first default layout.
func withPrinting(mode string) option {
	return func(h *harness) {
		h.Config.Print.Enabled = true
		h.Config.Print.PrinterName = h.Faker.LetterN(8)
		h.Config.Print.LayoutID = defaultLayoutIDs[0]
		h.Config.Print.SubmissionMode = mode
		h.Commands.answer("lpstat", lpstatLine(h.Config.Print.PrinterName, "idle"), nil)
	}
}

// lpstatJobs is how lpstat -o lists count jobs of a queue.
func lpstatJobs(name string, count int) string {
	var b strings.Builder
	for i := range count {
		_, _ = fmt.Fprintf(&b, "%s-%d          photopi        1024   Mon 12 Oct 2026 09:30:00 CEST\n", name, i+1)
	}
	return b.String()
}

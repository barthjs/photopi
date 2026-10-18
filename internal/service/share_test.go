package service

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"image/color"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/i18n"
	"github.com/barthjs/photopi/internal/mailer"
	"github.com/barthjs/photopi/internal/media"
	"github.com/barthjs/photopi/internal/share"
	"github.com/barthjs/photopi/internal/store"
)

// fakeTransport records sent mails instead of delivering them.
type fakeTransport struct {
	mu       sync.Mutex
	messages []mailer.Message
	err      error
}

// Send records the message and returns the prepared error.
func (r *fakeTransport) Send(_ context.Context, msg mailer.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages = append(r.messages, msg)
	return r.err
}

// fakeProvider records uploads instead of reaching a cloud.
type fakeProvider struct {
	mu      sync.Mutex
	folder  string
	files   []share.File
	uploads int
	url     string
	err     error
	testErr error
	started chan struct{}
	release chan struct{}
}

// TestConnection returns the prepared error.
func (f *fakeProvider) TestConnection(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.testErr
}

// Upload records the folder and the files, after waiting for release if started is set.
func (f *fakeProvider) Upload(_ context.Context, folder string, files []share.File) (string, error) {
	if f.started != nil {
		close(f.started)
		<-f.release
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.folder = folder
	f.files = files
	f.uploads++
	return f.url, f.err
}

// shareFixture is a share service with a recording mail transport and a fake cloud provider.
type shareFixture struct {
	svc       *ShareService
	storage   media.Storage
	transport *fakeTransport
	provider  *fakeProvider
}

// shareOptions changes the dependencies of a shareFixture.
type shareOptions struct {
	provider     share.Provider
	printEnabled bool
	// layoutID is the layout the collages are rendered with, a stored layout if empty.
	layoutID    string
	mailErr     error
	selfBaseURL string
}

func TestShareService_ListSharedFiles(t *testing.T) {
	t.Parallel()

	t.Run("returns the files of the run as signed URLs, the collage last", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		f := newShareFixture(t, shareOptions{
			selfBaseURL:  faker.URL(),
			printEnabled: true,
		})
		run := faker.IntRange(1, 1000)
		f.addCaptures(t, run)

		urls, err := f.svc.ListSharedFiles(t.Context(), run)

		require.NoError(t, err)
		require.Len(t, urls, 3)
		assert.Contains(t, urls[0], "x_"+pad(run)+"-01_")
		assert.Contains(t, urls[2], "-collage-1_")
		for _, url := range urls {
			assert.Contains(t, url, "signature=")
		}
	})

	t.Run("returns not found for an unknown run", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		f := newShareFixture(t, shareOptions{
			selfBaseURL: faker.URL(),
		})
		run := faker.IntRange(1, 1000)
		f.addCaptures(t, run)

		_, err := f.svc.ListSharedFiles(t.Context(), run+1)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestShareService_ListSharedFilePaths(t *testing.T) {
	t.Parallel()

	t.Run("returns the storage paths of the files of the run, the collage last", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		f := newShareFixture(t, shareOptions{
			selfBaseURL:  faker.URL(),
			printEnabled: true,
		})
		run := faker.IntRange(1, 1000)
		f.addCaptures(t, run)

		paths, err := f.svc.ListSharedFilePaths(t.Context(), run)

		require.NoError(t, err)
		require.Len(t, paths, 3)
		assert.Contains(t, paths[0], "x_"+pad(run)+"-01_")
		assert.Contains(t, paths[2], "-collage-1_")
	})

	t.Run("returns not found for an unknown run", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		f := newShareFixture(t, shareOptions{
			selfBaseURL: faker.URL(),
		})
		run := faker.IntRange(1, 1000)
		f.addCaptures(t, run)

		_, err := f.svc.ListSharedFilePaths(t.Context(), run+1)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestShareService_GetEmail(t *testing.T) {
	t.Parallel()

	t.Run("returns no state before an email was started", func(t *testing.T) {
		t.Parallel()

		f := newShareFixture(t, shareOptions{})
		run := apitest.Faker(t).IntRange(1, 1000)
		f.addCaptures(t, run)

		_, found := f.svc.GetEmail(run)

		assert.False(t, found, "no email was started yet")
	})
}

func TestShareService_GetCloudUpload(t *testing.T) {
	t.Parallel()

	t.Run("returns no state before an upload was started", func(t *testing.T) {
		t.Parallel()

		f := newShareFixture(t, shareOptions{})
		run := apitest.Faker(t).IntRange(1, 1000)
		f.addCaptures(t, run)

		_, found := f.svc.GetCloudUpload(run)

		assert.False(t, found, "no upload was started yet")
	})
}

func TestShareService_EmailEnabled(t *testing.T) {
	t.Parallel()

	t.Run("reports email sharing", func(t *testing.T) {
		t.Parallel()

		f := newShareFixture(t, shareOptions{})

		enabled := f.svc.EmailEnabled()

		assert.True(t, enabled)
	})
}

func TestShareService_CloudEnabled(t *testing.T) {
	t.Parallel()

	t.Run("reports a cloud provider", func(t *testing.T) {
		t.Parallel()

		f := newShareFixture(t, shareOptions{})

		enabled := f.svc.CloudEnabled()

		assert.True(t, enabled)
	})

	t.Run("reports the self share without a provider", func(t *testing.T) {
		t.Parallel()

		f := newShareFixture(t, shareOptions{
			provider:    newNoShareProvider(t),
			selfBaseURL: apitest.Faker(t).URL(),
		})

		enabled := f.svc.CloudEnabled()

		assert.True(t, enabled, "the self share needs no provider")
	})

	t.Run("reports no cloud sharing without a provider", func(t *testing.T) {
		t.Parallel()

		f := newShareFixture(t, shareOptions{
			provider: newNoShareProvider(t),
		})

		enabled := f.svc.CloudEnabled()

		assert.False(t, enabled)
	})
}

// newNoShareProvider returns the provider that disables cloud sharing.
func newNoShareProvider(t *testing.T) share.Provider {
	t.Helper()

	none, err := share.New(share.Options{
		Driver: share.DriverNone,
	})
	require.NoError(t, err)
	return none
}

func TestShareService_RunHasCaptures(t *testing.T) {
	t.Parallel()

	t.Run("reports a run with captures", func(t *testing.T) {
		t.Parallel()

		f := newShareFixture(t, shareOptions{})
		run := apitest.Faker(t).IntRange(1, 1000)
		f.addCaptures(t, run)

		has, err := f.svc.RunHasCaptures(t.Context(), run)

		require.NoError(t, err)
		assert.True(t, has)
	})

	t.Run("reports a run without captures", func(t *testing.T) {
		t.Parallel()

		f := newShareFixture(t, shareOptions{})
		run := apitest.Faker(t).IntRange(1, 1000)
		f.addCaptures(t, run)

		has, err := f.svc.RunHasCaptures(t.Context(), run+1)

		require.NoError(t, err)
		assert.False(t, has)
	})
}

func TestShareService_BeginEmail(t *testing.T) {
	t.Parallel()

	t.Run("marks the email as pending", func(t *testing.T) {
		t.Parallel()

		f := newShareFixture(t, shareOptions{})
		run := apitest.Faker(t).IntRange(1, 1000)

		begun := f.svc.BeginEmail(run)

		require.True(t, begun)
		state, found := f.svc.GetEmail(run)
		require.True(t, found)
		assert.Equal(t, JobPending, state.Status)
	})

	t.Run("rejects a second email while the first is in flight", func(t *testing.T) {
		t.Parallel()

		f := newShareFixture(t, shareOptions{})
		run := apitest.Faker(t).IntRange(1, 1000)
		require.True(t, f.svc.BeginEmail(run))

		begun := f.svc.BeginEmail(run)

		assert.False(t, begun, "an email in flight blocks a second one")
	})
}

func TestShareService_RunEmail(t *testing.T) {
	t.Parallel()

	t.Run("attaches every capture and records the delivery", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		f := newShareFixture(t, shareOptions{})
		run := faker.IntRange(1, 1000)
		recipient := faker.Email()
		captures := f.addCaptures(t, run)
		require.True(t, f.svc.BeginEmail(run))

		err := f.svc.RunEmail(t.Context(), run, recipient)

		require.NoError(t, err)
		state, _ := f.svc.GetEmail(run)
		assert.Equal(t, JobState{
			Status: JobDone,
		}, state)
		require.Len(t, f.transport.messages, 1)
		msg := f.transport.messages[0]
		assert.Equal(t, recipient, msg.To)
		assert.Equal(t, []mailer.Attachment{
			{
				Filename: "x_" + pad(run) + "-01_20260101-000000.jpg",
				Data:     captures[0],
			},
			{
				Filename: "x_" + pad(run) + "-02_20260101-000000.jpg",
				Data:     captures[1],
			},
		}, msg.Attachments)
	})

	t.Run("adds the print collage when printing is enabled", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		f := newShareFixture(t, shareOptions{
			printEnabled: true,
		})
		run := faker.IntRange(1, 1000)
		f.addCaptures(t, run)

		err := f.svc.RunEmail(t.Context(), run, faker.Email())

		require.NoError(t, err)
		attachments := f.transport.messages[0].Attachments
		require.Len(t, attachments, 3)
		assert.True(t, strings.HasPrefix(attachments[2].Filename, "x_"+pad(run)+"-collage-1_"), attachments[2].Filename)
		decodeJPEG(t, attachments[2].Data)
	})

	t.Run("leaves the collage out while printing is disabled", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		f := newShareFixture(t, shareOptions{
			printEnabled: false,
		})
		run := faker.IntRange(1, 1000)
		f.addCaptures(t, run)

		err := f.svc.RunEmail(t.Context(), run, faker.Email())

		require.NoError(t, err)
		assert.Len(t, f.transport.messages[0].Attachments, 2)
	})

	t.Run("sends the captures when the collage cannot be rendered", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		f := newShareFixture(t, shareOptions{
			printEnabled: true,
			layoutID:     faker.LetterN(8),
		})
		run := faker.IntRange(1, 1000)
		f.addCaptures(t, run)

		err := f.svc.RunEmail(t.Context(), run, faker.Email())

		require.NoError(t, err)
		assert.Len(t, f.transport.messages[0].Attachments, 2)
	})

	t.Run("returns not found for a run without captures", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		f := newShareFixture(t, shareOptions{})

		err := f.svc.RunEmail(t.Context(), faker.IntRange(1, 1000), faker.Email())

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("records a failed delivery", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		mailErr := errors.New(faker.LoremIpsumSentence(6))
		f := newShareFixture(t, shareOptions{
			mailErr: mailErr,
		})
		run := faker.IntRange(1, 1000)
		f.addCaptures(t, run)
		f.svc.BeginEmail(run)

		err := f.svc.RunEmail(t.Context(), run, faker.Email())

		assert.ErrorIs(t, err, mailErr)
		state, _ := f.svc.GetEmail(run)
		assert.Equal(t, JobState{
			Status: JobFailed,
			Reason: JobReasonFailed,
		}, state)
	})

	t.Run("records an unreachable mail server", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		f := newShareFixture(t, shareOptions{
			mailErr: &net.OpError{
				Op:  "dial",
				Err: errors.New(faker.LoremIpsumSentence(6)),
			},
		})
		run := faker.IntRange(1, 1000)
		f.addCaptures(t, run)
		f.svc.BeginEmail(run)

		err := f.svc.RunEmail(t.Context(), run, faker.Email())

		require.Error(t, err)
		state, _ := f.svc.GetEmail(run)
		assert.Equal(t, JobReasonUnreachable, state.Reason)
	})
}

func TestShareService_BeginCloudUpload(t *testing.T) {
	t.Parallel()

	t.Run("marks the upload as pending", func(t *testing.T) {
		t.Parallel()

		f := newShareFixture(t, shareOptions{})
		run := apitest.Faker(t).IntRange(1, 1000)
		f.addCaptures(t, run)

		begun := f.svc.BeginCloudUpload(run)

		require.True(t, begun)
		state, found := f.svc.GetCloudUpload(run)
		require.True(t, found)
		assert.Equal(t, JobPending, state.Status)
	})

	t.Run("starts only one upload per run at a time", func(t *testing.T) {
		t.Parallel()

		// The sequence of calls is the behavior under test: an upload in flight blocks only its own run
		faker := apitest.Faker(t)
		provider := &fakeProvider{
			url:     faker.URL(),
			started: make(chan struct{}),
			release: make(chan struct{}),
		}
		f := newShareFixture(t, shareOptions{
			provider: provider,
		})
		run := faker.IntRange(1, 1000)
		f.addCaptures(t, run)
		require.True(t, f.svc.BeginCloudUpload(run))
		done := make(chan error)
		go func() { done <- f.svc.RunCloudUpload(t.Context(), run) }()
		<-provider.started

		assert.False(t, f.svc.BeginCloudUpload(run), "an upload in flight blocks a second one")
		assert.True(t, f.svc.BeginCloudUpload(run+1), "other runs are independent")

		close(provider.release)
		require.NoError(t, <-done)
		assert.True(t, f.svc.BeginCloudUpload(run), "a finished upload can be started again")
	})
}

func TestShareService_RunCloudUpload(t *testing.T) {
	t.Parallel()

	t.Run("uploads the run into its folder below the event and records the link", func(t *testing.T) {
		t.Parallel()

		f := newShareFixture(t, shareOptions{})
		run := apitest.Faker(t).IntRange(1, 1000)
		f.addCaptures(t, run)
		require.True(t, f.svc.BeginCloudUpload(run))

		err := f.svc.RunCloudUpload(t.Context(), run)

		require.NoError(t, err)
		state, _ := f.svc.GetCloudUpload(run)
		assert.Equal(t, JobState{
			Status: JobDone,
			URL:    f.provider.url,
		}, state)
		assert.Equal(t, "summer-party/"+pad(run), f.provider.folder)
		assert.Len(t, f.provider.files, 2)
	})

	t.Run("shares a link to the public page without uploading", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		baseURL := faker.URL()
		f := newShareFixture(t, shareOptions{
			selfBaseURL: baseURL + "/",
		})
		run := faker.IntRange(1, 1000)
		f.addCaptures(t, run)
		f.svc.BeginCloudUpload(run)

		err := f.svc.RunCloudUpload(t.Context(), run)

		require.NoError(t, err)
		state, _ := f.svc.GetCloudUpload(run)
		require.Equal(t, JobDone, state.Status)
		assert.True(t, strings.HasPrefix(state.URL, baseURL+"/s/"), state.URL)
		assert.Zero(t, f.provider.uploads)
		token := strings.TrimPrefix(state.URL, baseURL+"/s/")
		resolved, ok := f.svc.ResolveSelfShareToken(token, time.Now())
		require.True(t, ok)
		assert.Equal(t, run, resolved)
	})

	t.Run("uploads again once the cloud is reachable", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		provider := &fakeProvider{
			url: faker.URL(),
			testErr: &net.OpError{
				Op:  "dial",
				Err: errors.New(faker.LoremIpsumSentence(6)),
			},
		}
		f := newShareFixture(t, shareOptions{
			provider: provider,
		})
		run := faker.IntRange(1, 1000)
		f.addCaptures(t, run)
		f.svc.checkCloud(t.Context())
		provider.mu.Lock()
		provider.testErr = nil
		provider.mu.Unlock()
		f.svc.checkCloud(t.Context())
		f.svc.BeginCloudUpload(run)

		err := f.svc.RunCloudUpload(t.Context(), run)

		require.NoError(t, err, "the upload works again once the cloud is reachable")
	})

	t.Run("records a failed upload", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		uploadErr := errors.New(faker.LoremIpsumSentence(6))
		f := newShareFixture(t, shareOptions{
			provider: &fakeProvider{
				err: uploadErr,
			},
		})
		run := faker.IntRange(1, 1000)
		f.addCaptures(t, run)
		f.svc.BeginCloudUpload(run)

		err := f.svc.RunCloudUpload(t.Context(), run)

		assert.ErrorIs(t, err, uploadErr)
		state, _ := f.svc.GetCloudUpload(run)
		assert.Equal(t, JobState{
			Status: JobFailed,
			Reason: JobReasonFailed,
		}, state)
	})

	t.Run("records a run without captures as failed", func(t *testing.T) {
		t.Parallel()

		f := newShareFixture(t, shareOptions{})
		run := apitest.Faker(t).IntRange(1, 1000)
		f.svc.BeginCloudUpload(run)

		err := f.svc.RunCloudUpload(t.Context(), run)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
		state, _ := f.svc.GetCloudUpload(run)
		assert.Equal(t, JobFailed, state.Status)
	})

	t.Run("fails right away while the cloud is unreachable", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		provider := &fakeProvider{
			url: faker.URL(),
			testErr: &net.OpError{
				Op:  "dial",
				Err: errors.New(faker.LoremIpsumSentence(6)),
			},
		}
		f := newShareFixture(t, shareOptions{
			provider: provider,
		})
		run := faker.IntRange(1, 1000)
		f.addCaptures(t, run)
		f.svc.checkCloud(t.Context())
		f.svc.BeginCloudUpload(run)

		err := f.svc.RunCloudUpload(t.Context(), run)

		require.Error(t, err)
		state, _ := f.svc.GetCloudUpload(run)
		assert.Equal(t, JobReasonUnreachable, state.Reason)
		assert.Zero(t, provider.uploads, "nothing is uploaded while the cloud is unreachable")
	})

	t.Run("marks the cloud unreachable after an unreachable upload", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		provider := &fakeProvider{
			err: &net.OpError{
				Op:  "dial",
				Err: errors.New(faker.LoremIpsumSentence(6)),
			},
		}
		f := newShareFixture(t, shareOptions{
			provider: provider,
		})
		run := faker.IntRange(1, 1000)
		f.addCaptures(t, run)
		f.svc.BeginCloudUpload(run)
		require.Error(t, f.svc.RunCloudUpload(t.Context(), run))
		f.svc.BeginCloudUpload(run)

		err := f.svc.RunCloudUpload(t.Context(), run)

		assert.ErrorIs(t, err, errCloudUnreachable)
		assert.Equal(t, 1, provider.uploads, "the next upload waits for the cloud to be reachable")
	})
}

func TestShareService_checkCloud(t *testing.T) {
	t.Parallel()

	t.Run("keeps the cloud reachable for rejected credentials", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		provider := &fakeProvider{
			url:     faker.URL(),
			testErr: errors.New(faker.LoremIpsumSentence(6)),
		}
		f := newShareFixture(t, shareOptions{
			provider: provider,
		})

		f.svc.checkCloud(t.Context())

		assert.False(t, f.svc.cloudUnreachable.Load(), "rejected credentials do not count as unreachable")
	})
}

func TestShareService_ResolveSelfShareToken(t *testing.T) {
	t.Parallel()

	t.Run("returns the run until the link expires", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		f := newShareFixture(t, shareOptions{
			selfBaseURL: faker.URL(),
		})
		run := faker.IntRange(1, 1000)
		token := selfShareToken(t, f.svc, run, time.Now())

		_, ok := f.svc.ResolveSelfShareToken(token, time.Now().Add(selfShareLinkTTL-time.Minute))

		assert.True(t, ok)
	})

	t.Run("rejects an expired token", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		f := newShareFixture(t, shareOptions{
			selfBaseURL: faker.URL(),
		})
		run := faker.IntRange(1, 1000)
		token := selfShareToken(t, f.svc, run, time.Now())

		_, ok := f.svc.ResolveSelfShareToken(token, time.Now().Add(selfShareLinkTTL+time.Minute))

		assert.False(t, ok)
	})

	t.Run("rejects a tampered token", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		f := newShareFixture(t, shareOptions{
			selfBaseURL: faker.URL(),
		})
		token := selfShareToken(t, f.svc, faker.IntRange(1, 1000), time.Now())
		expires, signed, _ := strings.Cut(token, ".")

		for _, tampered := range []string{
			expires + "0." + signed,
			"9" + expires + "." + signed,
			signed,
			"",
		} {
			_, ok := f.svc.ResolveSelfShareToken(tampered, time.Now())

			assert.False(t, ok, tampered)
		}
	})

	t.Run("rejects every token while the self share is disabled", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		f := newShareFixture(t, shareOptions{
			selfBaseURL: faker.URL(),
		})
		token := selfShareToken(t, f.svc, faker.IntRange(1, 1000), time.Now())
		f.svc.options.SelfBaseURL = ""

		_, ok := f.svc.ResolveSelfShareToken(token, time.Now())

		assert.False(t, ok)
	})
}

// selfShareToken returns the token of the self share link of run, signed at now.
func selfShareToken(t *testing.T, svc *ShareService, run int, now time.Time) string {
	t.Helper()

	link := svc.selfShareURL(run, now)
	_, token, found := strings.Cut(link, "/s/")
	require.True(t, found, link)
	return token
}

func TestShareService_ZipSharedFiles(t *testing.T) {
	t.Parallel()

	t.Run("returns an archive of the files of the run", func(t *testing.T) {
		t.Parallel()

		faker := apitest.Faker(t)
		f := newShareFixture(t, shareOptions{
			selfBaseURL: faker.URL(),
		})
		run := faker.IntRange(1, 1000)
		f.addCaptures(t, run)
		paths, err := f.svc.ListSharedFilePaths(t.Context(), run)
		require.NoError(t, err)
		var buf bytes.Buffer

		err = f.svc.ZipSharedFiles(t.Context(), &buf, paths)

		require.NoError(t, err)
		reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		require.NoError(t, err)
		assert.Len(t, reader.File, 2)
	})
}

// newShareFixture builds a share service with email sharing enabled, local storage, and two captures per run.
func newShareFixture(t *testing.T, options shareOptions) shareFixture {
	t.Helper()

	faker := apitest.Faker(t)
	storage := newTestStorage(t)
	bundle, err := i18n.New()
	require.NoError(t, err)
	transport := &fakeTransport{
		err: options.mailErr,
	}
	mail, err := mailer.New(zerolog.Nop(), mailer.Options{
		FromAddress: faker.Email(),
		Transport:   transport,
	}, bundle)
	require.NoError(t, err)

	provider, _ := options.provider.(*fakeProvider)
	if options.provider == nil {
		provider = &fakeProvider{
			url: faker.URL(),
		}
		options.provider = provider
	}
	layouts := store.NewLayoutStore(storage)
	require.NoError(t, layouts.Save(t.Context(), twoSlotLayout(1).ID, twoSlotLayout(1)))
	if options.layoutID == "" {
		options.layoutID = twoSlotLayout(1).ID
	}

	runs := store.NewRunStore(storage)
	captures := store.NewCaptureStore(storage)
	capture := newCaptureServiceOn(storage, CaptureOptions{
		ImagesPerRun: 2,
	})
	printService := NewPrintService(zerolog.Nop(), PrintOptions{
		LayoutID:   options.layoutID,
		EventName:  "Summer Party",
		FilePrefix: "x",
	}, &fakeCommands{}, runs, captures, store.NewCollageStore(storage), layouts)
	svc := NewShareService(zerolog.Nop(), ShareOptions{
		AppName:        faker.AppName(),
		Language:       "en",
		EventName:      "Summer Party",
		EmailEnabled:   true,
		EmailSubject:   faker.LoremIpsumSentence(6),
		PrintEnabled:   options.printEnabled,
		SelfBaseURL:    options.selfBaseURL,
		SelfSigningKey: []byte(faker.Password(true, true, true, false, false, 32)),
	}, mail, options.provider, runs, capture, printService)
	return shareFixture{
		svc:       svc,
		storage:   storage,
		transport: transport,
		provider:  provider,
	}
}

// addCaptures stores two captures of random colors for run and returns their content in order.
func (f shareFixture) addCaptures(t *testing.T, run int) [][]byte {
	t.Helper()

	faker := apitest.Faker(t)
	captures := [][]byte{
		testJPEG(t, 8, 8, randomColor(faker)),
		testJPEG(t, 8, 8, randomColor(faker)),
	}
	putFile(t, f.storage, "captures/party/x_"+pad(run)+"-01_20260101-000000.jpg", captures[0])
	putFile(t, f.storage, "captures/party/x_"+pad(run)+"-02_20260101-000000.jpg", captures[1])
	return captures
}

// randomColor returns an opaque color from the faker.
func randomColor(faker *gofakeit.Faker) color.Color {
	return color.RGBA{
		R: faker.Uint8(),
		G: faker.Uint8(),
		B: faker.Uint8(),
		A: 255,
	}
}

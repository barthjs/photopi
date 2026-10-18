package api_test

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/api"
	"github.com/barthjs/photopi/internal/config"
	"github.com/barthjs/photopi/internal/share"
)

func TestPublicShareAccess(t *testing.T) {
	t.Parallel()

	// The page needs no session, only a link with a valid signature
	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/share/public/{token}",
			arrange: func(h *harness) (string, any) {
				return h.publicPath(h.completeRun()), nil
			},
			statuses: map[*clientKind]int{
				clientOtherGuest: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/share/public/{token}/zip",
			arrange: func(h *harness) (string, any) {
				return h.publicPath(h.completeRun()) + "/zip", nil
			},
			statuses: map[*clientKind]int{
				clientOtherGuest: http.StatusOK,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/share/public/{forged token}",
			arrange: func(h *harness) (string, any) {
				return forgedPublicPath(h), nil
			},
			statuses: map[*clientKind]int{
				clientOtherGuest: http.StatusNotFound,
			},
		},
		{
			method: http.MethodGet,
			path:   "/api/share/public/{forged token}/zip",
			arrange: func(h *harness) (string, any) {
				return forgedPublicPath(h) + "/zip", nil
			},
			statuses: map[*clientKind]int{
				clientOtherGuest: http.StatusNotFound,
			},
		},
	}, withSelfShare())
}

// forgedPublicPath shares a run and returns the path of its public share page with a forged signature.
func forgedPublicPath(h *harness) string {
	h.t.Helper()

	valid := h.publicPath(h.completeRun())
	expires, _, _ := strings.Cut(strings.TrimPrefix(valid, "/api/share/public/"), ".")
	return "/api/share/public/" + expires + "." + h.Faker.DigitN(6) + "." + h.Faker.HexUint(128)[2:]
}

func TestPublicShareGetHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the signed file URLs", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withSelfShare())
		run := h.startRun()
		photo := newJPEG(t, h.Faker)
		h.uploadCapture(run, photo).AssertCreated()

		shared := h.newClient().GET(h.publicPath(run)).AssertOK().Decode[api.PublicShareResponse]()

		require.Len(t, shared.ImageURLs, 1)
		assert.True(t, strings.HasPrefix(shared.ImageURLs[0], api.MediaPath+"/"), shared.ImageURLs[0])
		file := h.newClient().GET(shared.ImageURLs[0]).AssertOK()
		assert.Equal(t, photo, file.Body, "the guest's phone loads the file through the signed URL")
	})

	t.Run("returns not found for a malformed token", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withSelfShare())

		response := h.Client.GET("/api/share/public/" + h.Faker.LetterN(20))

		response.AssertNotFound()
	})

	t.Run("returns not found without the self share", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withSelfShare())
		publicPath := h.publicPath(h.completeRun())
		other := newHarness(t, func(other *harness) {
			other.Config.App.Secret = h.Config.App.Secret
		})

		response := other.Client.GET(publicPath)

		response.AssertNotFound()
	})

	t.Run("returns not found for an unknown run", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withSelfShare())
		run := h.completeRun()
		publicPath := h.publicPath(run)
		h.asAdmin().DELETE(galleryPath(run)).AssertNoContent()

		response := h.newClient().GET(publicPath)

		response.AssertNotFound()
	})

	t.Run("rejects a file URL with a forged signature", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withSelfShare())
		signed := h.signedFileURL(h.completeRun())
		query := signed.Query()
		query.Set("signature", h.Faker.LetterN(43))
		signed.RawQuery = query.Encode()

		response := h.Client.GET(signed.String())

		response.AssertForbidden()
	})

	t.Run("rejects a file URL without a signature", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withSelfShare())
		signed := h.signedFileURL(h.completeRun())

		response := h.Client.GET(signed.Path)

		response.AssertForbidden()
	})
}

// signedFileURL returns the signed URL of the first file on the public share page of run.
func (h *harness) signedFileURL(run int) *url.URL {
	h.t.Helper()

	shared := h.Client.GET(h.publicPath(run)).AssertOK().Decode[api.PublicShareResponse]()
	require.NotEmpty(h.t, shared.ImageURLs)
	signed, err := url.Parse(shared.ImageURLs[0])
	require.NoError(h.t, err)
	return signed
}

func TestPublicShareZipHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns the captures", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withSelfShare())
		run := h.completeRun()

		response := h.newClient().GET(h.publicPath(run) + "/zip")

		response.AssertOK().AssertHeader("Content-Type", "application/zip")
		archive, err := zip.NewReader(bytes.NewReader(response.Body), int64(len(response.Body)))
		require.NoError(t, err)
		assert.Len(t, archive.File, h.Config.Capture.ImagesPerRun)
	})

	t.Run("returns not found for an unknown run", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withSelfShare())
		run := h.completeRun()
		publicPath := h.publicPath(run)
		h.asAdmin().DELETE(galleryPath(run)).AssertNoContent()

		response := h.newClient().GET(publicPath + "/zip")

		response.AssertNotFound()
	})
}

// withSelfShare shares runs through the public share page of the application.
func withSelfShare() option {
	return withConfig(func(cfg *config.Config) {
		cfg.Share.Cloud.Driver = share.DriverSelf
		cfg.Share.Cloud.Self.BaseURL = "http://booth.test"
	})
}

// publicPath shares run through the self share as its guest would
// and returns the API path of the public share page the link points to.
func (h *harness) publicPath(run int) string {
	h.t.Helper()

	h.Client.POST(sharePath(run)+"/cloud-upload", nil).AssertStatus(http.StatusAccepted)
	link, err := url.Parse(h.awaitJob(sharePath(run) + "/cloud-upload").URL)
	require.NoError(h.t, err)

	token, found := strings.CutPrefix(link.Path, "/s/")
	require.True(h.t, found, link.String())
	return "/api/share/public/" + token
}

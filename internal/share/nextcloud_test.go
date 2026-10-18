package share_test

import (
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/apitest"
	"github.com/barthjs/photopi/internal/share"
)

const (
	ncUser     = "booth"
	ncPassword = "secret"
)

func TestNextcloudProvider_TestConnection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   int
		password string
		wantErr  string
	}{
		{
			name:     "accepts a multi status",
			status:   http.StatusMultiStatus,
			password: ncPassword,
		},
		{
			name:     "accepts an ok status",
			status:   http.StatusOK,
			password: ncPassword,
		},
		{
			name:     "rejects wrong credentials",
			status:   http.StatusMultiStatus,
			password: "wrong",
			wantErr:  "rejected the credentials (status 401)",
		},
		{
			name:     "rejects a forbidden account",
			status:   http.StatusForbidden,
			password: ncPassword,
			wantErr:  "rejected the credentials (status 403)",
		},
		{
			name:     "rejects an unexpected status",
			status:   http.StatusInternalServerError,
			password: ncPassword,
			wantErr:  "unexpected status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake, server := newFakeNextcloud(t)
			fake.propfindStatus = tt.status

			err := newNextcloud(t, server.URL, "", tt.password).TestConnection(t.Context())

			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}

	t.Run("rejects an unreachable server", func(t *testing.T) {
		t.Parallel()

		_, server := newFakeNextcloud(t)
		server.Close()

		err := newNextcloud(t, server.URL, "", ncPassword).TestConnection(t.Context())

		require.Error(t, err)
		assert.ErrorContains(t, err, "connection failed")
	})

	t.Run("rejects an invalid url", func(t *testing.T) {
		t.Parallel()

		provider := newNextcloud(t, "http://bad host", "", ncPassword)

		err := provider.TestConnection(t.Context())

		require.Error(t, err)
	})
}

func TestNextcloudProvider_Upload(t *testing.T) {
	t.Parallel()

	t.Run("uploads the files into new folders and shares them", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		fake, server := newFakeNextcloud(t)
		files := []share.File{
			{
				Name: "1.jpg",
				Data: []byte(f.LoremIpsumSentence(6)),
			},
			{
				Name: "2.jpg",
				Data: []byte(f.LoremIpsumSentence(6)),
			},
		}

		shareURL, err := newNextcloud(t, server.URL, "/photobooth/2026/", ncPassword).Upload(t.Context(), "/run-7/", files)

		require.NoError(t, err)
		assert.Equal(t, "https://cloud.example.test/s/abc", shareURL)
		assert.Equal(t, []string{"/photobooth", "/photobooth/2026", "/photobooth/2026/run-7"}, fake.folders)
		assert.Equal(t, map[string][]byte{
			"/photobooth/2026/run-7/1.jpg": files[0].Data,
			"/photobooth/2026/run-7/2.jpg": files[1].Data,
		}, fake.files)
		require.Len(t, fake.shares, 1)
		assert.Equal(t, "/photobooth/2026/run-7", fake.shares[0].Get("path"))
		assert.Equal(t, "3", fake.shares[0].Get("shareType"), "a public link")
		assert.Equal(t, "1", fake.shares[0].Get("permissions"), "read only")
	})

	t.Run("reuses the existing link", func(t *testing.T) {
		t.Parallel()

		fake, server := newFakeNextcloud(t)
		provider := newNextcloud(t, server.URL, "", ncPassword)
		files := []share.File{
			{
				Name: "a.jpg",
				Data: []byte("a"),
			},
		}
		first, err := provider.Upload(t.Context(), "run-5", files)
		require.NoError(t, err)

		second, err := provider.Upload(t.Context(), "run-5", files)

		require.NoError(t, err)
		assert.Equal(t, first, second)
		assert.Len(t, fake.shares, 1, "a repeated upload creates no further link")
	})

	t.Run("uploads without a root folder", func(t *testing.T) {
		t.Parallel()

		fake, server := newFakeNextcloud(t)

		_, err := newNextcloud(t, server.URL, "", ncPassword).Upload(t.Context(), "run-8", []share.File{
			{
				Name: "a.jpg",
				Data: []byte("a"),
			},
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"/run-8"}, fake.folders)
		assert.Contains(t, fake.files, "/run-8/a.jpg")
		assert.Equal(t, "/run-8", fake.shares[0].Get("path"))
	})

	t.Run("keeps existing folders", func(t *testing.T) {
		t.Parallel()

		fake, server := newFakeNextcloud(t)
		fake.folders = []string{"/photobooth"}

		_, err := newNextcloud(t, server.URL, "photobooth", ncPassword).Upload(t.Context(), "run-9", nil)

		require.NoError(t, err)
		assert.Equal(t, []string{"/photobooth", "/photobooth/run-9"}, fake.folders)
	})

	failureTests := []struct {
		name      string
		prepare   func(*fakeNextcloud)
		password  string
		wantErr   string
		wantShare bool
	}{
		{
			name:     "rejects a refused file",
			prepare:  func(f *fakeNextcloud) { f.putStatus = http.StatusInsufficientStorage },
			password: ncPassword,
			wantErr:  `upload of "a.jpg" failed: unexpected status 507`,
		},
		{
			name:     "rejects wrong credentials",
			prepare:  func(*fakeNextcloud) {},
			password: "wrong",
			wantErr:  `could not create folder "run": unexpected status 401`,
		},
		{
			name: "rejects an error of the share api",
			prepare: func(f *fakeNextcloud) {
				f.shareBody = `{"ocs":{"meta":{"status":"failure","statuscode":404,"message":"Wrong path"}}}`
			},
			password:  ncPassword,
			wantErr:   "OCS API error: 404 Wrong path",
			wantShare: true,
		},
		{
			name:      "rejects a share response that is not json",
			prepare:   func(f *fakeNextcloud) { f.shareBody = "<html>" },
			password:  ncPassword,
			wantErr:   "decoding share response",
			wantShare: true,
		},
	}

	for _, tt := range failureTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake, server := newFakeNextcloud(t)
			tt.prepare(fake)

			_, err := newNextcloud(t, server.URL, "", tt.password).Upload(t.Context(), "run", []share.File{
				{
					Name: "a.jpg",
					Data: []byte("a"),
				},
			})

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
			assert.Equal(t, tt.wantShare, len(fake.shares) == 1)
		})
	}

	folderTests := []struct {
		name    string
		status  int
		wantErr string
	}{
		{
			name:    "rejects a conflict on a folder",
			status:  http.StatusConflict,
			wantErr: `could not create folder "photobooth": unexpected status 409`,
		},
		{
			name:    "rejects a server error on a folder",
			status:  http.StatusInternalServerError,
			wantErr: `could not create folder "photobooth": unexpected status 500`,
		},
	}

	for _, tt := range folderTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake, server := newFakeNextcloud(t)
			fake.mkcolStatus = tt.status

			_, err := newNextcloud(t, server.URL, "photobooth", ncPassword).Upload(t.Context(), "run", []share.File{
				{
					Name: "a.jpg",
				},
			})

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
			assert.Empty(t, fake.files, "nothing is uploaded into a missing folder")
			assert.Empty(t, fake.shares)
		})
	}

	t.Run("names the nested folder in the error", func(t *testing.T) {
		t.Parallel()

		fake, server := newFakeNextcloud(t)
		fake.folders = []string{"/photobooth"}
		fake.mkcolStatus = http.StatusForbidden

		_, err := newNextcloud(t, server.URL, "photobooth", ncPassword).Upload(t.Context(), "run-3", nil)

		require.Error(t, err)
		assert.ErrorContains(t, err, `could not create folder "photobooth/run-3": unexpected status 403`)
	})

	t.Run("rejects an unreachable server", func(t *testing.T) {
		t.Parallel()

		_, server := newFakeNextcloud(t)
		server.Close()

		_, err := newNextcloud(t, server.URL, "", ncPassword).Upload(t.Context(), "run", []share.File{
			{
				Name: "a.jpg",
			},
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, `could not create folder "run"`)
	})

	droppedTests := []struct {
		name    string
		method  string
		files   []share.File
		wantErr string
	}{
		{
			name:   "rejects a dropped connection during the upload",
			method: http.MethodPut,
			files: []share.File{
				{
					Name: "a.jpg",
				},
			},
			wantErr: `upload of "a.jpg" failed`,
		},
		{
			name:    "rejects a dropped connection during the share",
			method:  http.MethodPost,
			wantErr: "creating share",
		},
	}

	for _, tt := range droppedTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake, server := newFakeNextcloud(t)
			fake.dropMethod = tt.method

			_, err := newNextcloud(t, server.URL, "", ncPassword).Upload(t.Context(), "run", tt.files)

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}

	invalidURLTests := []struct {
		name  string
		files []share.File
	}{
		{
			name: "rejects an invalid url for files",
			files: []share.File{
				{
					Name: "a.jpg",
				},
			},
		},
		{
			name:  "rejects an invalid url without files",
			files: nil,
		},
	}

	for _, tt := range invalidURLTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := newNextcloud(t, "http://bad host", "", ncPassword)

			_, err := provider.Upload(t.Context(), "run", tt.files)

			require.Error(t, err)
		})
	}
}

// newFakeNextcloud starts a fake Nextcloud that accepts every request of the test user.
func newFakeNextcloud(t *testing.T) (*fakeNextcloud, *httptest.Server) {
	t.Helper()

	fake := &fakeNextcloud{
		files:          map[string][]byte{},
		propfindStatus: http.StatusMultiStatus,
		putStatus:      http.StatusCreated,
	}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	return fake, server
}

// newNextcloud returns a Nextcloud provider for the test user on baseURL.
func newNextcloud(t *testing.T, baseURL, folder, password string) share.Provider {
	t.Helper()

	provider, err := share.New(share.Options{
		Driver:            share.DriverNextcloud,
		NextcloudURL:      baseURL,
		NextcloudUsername: ncUser,
		NextcloudPassword: password,
		NextcloudFolder:   folder,
	})
	require.NoError(t, err)
	return provider
}

// fakeNextcloud is an in-memory Nextcloud that answers the WebDAV and OCS share requests the provider sends.
type fakeNextcloud struct {
	mu sync.Mutex

	folders  []string
	files    map[string][]byte
	shares   []url.Values
	requests []string

	propfindStatus int
	putStatus      int
	shareBody      string
	// mkcolStatus answers MKCOL for a missing folder with this status instead of creating it, 0 for the normal behavior.
	mkcolStatus int
	// dropMethod names a method whose requests are answered by closing the connection, to simulate a network failure.
	dropMethod string
}

// ServeHTTP checks the credentials and answers the request like Nextcloud, recording what it receives.
func (f *fakeNextcloud) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.requests = append(f.requests, r.Method+" "+r.URL.Path)

	user, password, ok := r.BasicAuth()
	if !ok || user != ncUser || password != ncPassword {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if r.Method == f.dropMethod {
		hijacker, ok := w.(http.Hijacker)
		if ok {
			conn, _, err := hijacker.Hijack()
			if err == nil {
				_ = conn.Close()
				return
			}
		}
	}

	const webdavPrefix = "/remote.php/dav/files/" + ncUser
	switch {
	case r.Method == "PROPFIND":
		w.WriteHeader(f.propfindStatus)
	case r.Method == "MKCOL":
		path := r.URL.Path[len(webdavPrefix):]
		if slices.Contains(f.folders, path) {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if f.mkcolStatus != 0 {
			w.WriteHeader(f.mkcolStatus)
			return
		}
		f.folders = append(f.folders, path)
		w.WriteHeader(http.StatusCreated)
	case r.Method == http.MethodPut:
		data, _ := io.ReadAll(r.Body)
		f.files[r.URL.Path[len(webdavPrefix):]] = data
		w.WriteHeader(f.putStatus)
	case r.Method == http.MethodGet && r.URL.Path == "/ocs/v2.php/apps/files_sharing/api/v1/shares":
		data := make([]map[string]any, 0)
		for _, created := range f.shares {
			if created.Get("path") == r.URL.Query().Get("path") {
				data = append(data, map[string]any{
					"share_type": 3,
					"url":        "https://cloud.example.test/s/abc",
				})
			}
		}
		_ = json.MarshalWrite(w, map[string]any{
			"ocs": map[string]any{
				"meta": map[string]any{
					"status":     "ok",
					"statuscode": 200,
				},
				"data": data,
			},
		})
	case r.Method == http.MethodPost && r.URL.Path == "/ocs/v2.php/apps/files_sharing/api/v1/shares":
		if r.Header.Get("OCS-APIRequest") != "true" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = r.ParseForm()
		f.shares = append(f.shares, r.PostForm)
		if f.shareBody != "" {
			_, _ = io.WriteString(w, f.shareBody)
			return
		}
		_ = json.MarshalWrite(w, map[string]any{
			"ocs": map[string]any{
				"meta": map[string]any{"status": "ok", "statuscode": 200},
				"data": map[string]any{"url": "https://cloud.example.test/s/abc"},
			},
		})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

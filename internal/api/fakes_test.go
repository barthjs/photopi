package api_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"sync"
	"testing"

	"github.com/brianvoe/gofakeit/v7"

	"github.com/barthjs/photopi/internal/mailer"
	"github.com/barthjs/photopi/internal/share"
	"github.com/barthjs/photopi/internal/update"
)

// fakeCamera returns generated JPEGs instead of driving a camera.
type fakeCamera struct {
	mu         sync.Mutex
	image      []byte
	err        error
	captures   int
	previewErr error
	endless    bool
	// previewsEnded counts the previews that returned.
	previewsEnded int
}

// newFakeCamera returns a fake camera that serves one generated JPEG.
func newFakeCamera(t testing.TB, f *gofakeit.Faker) *fakeCamera {
	return &fakeCamera{
		image: newJPEG(t, f),
	}
}

// StreamPreview writes the image as one frame of a multipart stream and then returns the prepared preview error,
// or writes it as frames until ctx ends or writing fails if the preview is endless.
func (c *fakeCamera) StreamPreview(ctx context.Context, w io.Writer, boundary string) error {
	c.mu.Lock()
	frame, previewErr, endless := c.image, c.previewErr, c.endless
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.previewsEnded++
		c.mu.Unlock()
	}()

	for {
		if _, err := fmt.Fprintf(w, "--%s\r\nContent-Type: image/jpeg\r\n\r\n%s\r\n", boundary, frame); err != nil {
			return err
		}
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		if !endless {
			return previewErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
}

// failPreview makes every following preview fail with err after its frame.
func (c *fakeCamera) failPreview(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.previewErr = err
}

// endedPreviews returns the number of previews that returned so far.
func (c *fakeCamera) endedPreviews() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.previewsEnded
}

// streamEndlessly makes every following preview write frames until the client leaves.
func (c *fakeCamera) streamEndlessly() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.endless = true
}

// Capture counts the capture and returns a copy of the image, or the prepared error.
func (c *fakeCamera) Capture(context.Context) (*[]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.captures++
	if c.err != nil {
		return nil, c.err
	}
	data := slices.Clone(c.image)
	return &data, nil
}

// fail makes every following capture fail with err.
func (c *fakeCamera) fail(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.err = err
}

// returnImage makes every following capture return data.
func (c *fakeCamera) returnImage(data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.image = data
}

// captureCount returns the number of captures so far.
func (c *fakeCamera) captureCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.captures
}

// fakeCloudProvider records uploads instead of sending them to a cloud.
type fakeCloudProvider struct {
	mu         sync.Mutex
	url        string
	testErr    error
	uploadErr  error
	folders    []string
	fileCounts []int
}

// TestConnection returns the prepared connection error.
func (p *fakeCloudProvider) TestConnection(context.Context) error {
	return p.testErr
}

// Upload records the folder and the number of files and returns the prepared URL or error.
func (p *fakeCloudProvider) Upload(_ context.Context, remoteFolder string, files []share.File) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.folders = append(p.folders, remoteFolder)
	p.fileCounts = append(p.fileCounts, len(files))
	if p.uploadErr != nil {
		return "", p.uploadErr
	}
	return p.url, nil
}

// uploads returns the number of files of every upload so far.
func (p *fakeCloudProvider) uploads() []int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.fileCounts)
}

// commandCall is one recorded program call.
type commandCall struct {
	Name string
	Args []string
	// File holds the content of a file passed as the last argument, read during the call,
	// since programs like lp receive a temporary file that is removed afterward.
	File []byte
}

// commandResult is what a faked program answers.
type commandResult struct {
	Output []byte
	Err    error
}

// fakeCommands records program calls instead of running them and answers with prepared results.
type fakeCommands struct {
	mu      sync.Mutex
	calls   []commandCall
	results map[string]commandResult
}

// newFakeCommands returns a fake without prepared results.
func newFakeCommands() *fakeCommands {
	return &fakeCommands{
		results: make(map[string]commandResult),
	}
}

// answer prepares the result of every following call of the program.
func (c *fakeCommands) answer(name string, output string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results[name] = commandResult{
		Output: []byte(output),
		Err:    err,
	}
}

// answerArg prepares the result of every following call of the program with firstArg as its first argument,
// such as lpstat -o, which takes precedence over the result prepared by answer.
func (c *fakeCommands) answerArg(name string, firstArg string, output string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results[name+" "+firstArg] = commandResult{
		Output: []byte(output),
		Err:    err,
	}
}

// Start records the call and returns the prepared error.
func (c *fakeCommands) Start(name string, args ...string) error {
	return c.record(name, args).Err
}

// Output records the call and returns the prepared result.
func (c *fakeCommands) Output(_ context.Context, _ []string, name string, args ...string) ([]byte, error) {
	result := c.record(name, args)
	return result.Output, result.Err
}

// CombinedOutput records the call and returns the prepared result.
func (c *fakeCommands) CombinedOutput(_ context.Context, _ []string, name string, args ...string) ([]byte, error) {
	result := c.record(name, args)
	return result.Output, result.Err
}

// record stores the call and returns the prepared result of the program.
func (c *fakeCommands) record(name string, args []string) commandResult {
	c.mu.Lock()
	defer c.mu.Unlock()

	call := commandCall{
		Name: name,
		Args: slices.Clone(args),
	}
	if len(args) > 0 {
		if data, err := os.ReadFile(args[len(args)-1]); err == nil {
			call.File = data
		}
	}
	c.calls = append(c.calls, call)
	if len(args) > 0 {
		if result, ok := c.results[name+" "+args[0]]; ok {
			return result
		}
	}
	return c.results[name]
}

// callsOf returns the recorded calls of one program.
func (c *fakeCommands) callsOf(name string) []commandCall {
	c.mu.Lock()
	defer c.mu.Unlock()

	var calls []commandCall
	for _, call := range c.calls {
		if call.Name == name {
			calls = append(calls, call)
		}
	}
	return calls
}

// count returns the number of recorded calls.
func (c *fakeCommands) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.calls)
}

// fakeMailTransport records messages instead of sending them.
type fakeMailTransport struct {
	mu       sync.Mutex
	messages []mailer.Message
	err      error
	held     chan struct{}
}

// Send records the message and returns the prepared error, after waiting for the release of a hold.
func (m *fakeMailTransport) Send(ctx context.Context, msg mailer.Message) error {
	m.mu.Lock()
	held := m.held
	m.mu.Unlock()
	if held != nil {
		select {
		case <-held:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.messages = append(m.messages, msg)
	return nil
}

// fail makes every further delivery fail with err.
func (m *fakeMailTransport) fail(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// hold makes every further delivery wait until the test ends.
func (m *fakeMailTransport) hold(t *testing.T) {
	t.Helper()

	m.mu.Lock()
	defer m.mu.Unlock()
	held := make(chan struct{})
	m.held = held
	t.Cleanup(func() { close(held) })
}

// sent returns the messages recorded so far.
func (m *fakeMailTransport) sent() []mailer.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.messages)
}

// newFakeUpdater returns an updater that asks a fake GitHub API, which reports latest as the latest release.
// Without an architecture, it never replaces the binary of the test.
func newFakeUpdater(t *testing.T, latest string) *update.Updater {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.MarshalWrite(w, map[string]any{
			"tag_name": latest,
			"html_url": "https://example.test/releases/" + latest,
			"assets":   []any{},
		})
	}))
	t.Cleanup(srv.Close)

	return update.New(update.Options{
		Repository: "owner/app",
		Binary:     "app",
		Version:    harnessVersion,
		APIURL:     srv.URL,
	})
}

// errFake is returned by fakes that are told to fail.
var errFake = errors.New("fake failure")

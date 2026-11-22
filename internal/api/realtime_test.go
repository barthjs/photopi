package api_test

import (
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/domain"
)

// TestRealtimeAccess checks the status of the WebSocket route for each kind of client. Every client may connect,
// so a request without a WebSocket upgrade fails alike for all of them.
func TestRealtimeAccess(t *testing.T) {
	t.Parallel()

	testAccess(t, []accessRoute{
		{
			method: http.MethodGet,
			path:   "/api/ws",
			statuses: map[*clientKind]int{
				clientGuest: http.StatusBadRequest,
				clientAdmin: http.StatusBadRequest,
			},
		},
	})
}

// TestRealtimeConnectHandler checks the WebSocket connection of a client over a real server.
func TestRealtimeConnectHandler(t *testing.T) {
	t.Parallel()

	t.Run("greets and sends the state of the dependencies on connect", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		conn := dialRealtime(t, h)

		hello := readData[domain.Hello](t, conn, domain.MessageHello)
		assert.Equal(t, harnessVersion, hello.Version)
		assert.NotEmpty(t, hello.Meta)
		checks := readData[map[string]string](t, conn, domain.MessageHealth)
		assert.Contains(t, checks, "camera")
	})

	t.Run("announces a failing camera", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, withDeviceCamera())
		h.Camera.failPreview(errFake)
		srv, client := newPreviewServer(t, h)
		conn := dialRealtime(t, h)
		readGreeting(t, conn)

		response, err := getPreview(t, srv, client)
		require.NoError(t, err)
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()

		checks := readData[map[string]string](t, conn, domain.MessageHealth)
		assert.Equal(t, "unreachable", checks["camera"])
	})
}

// dialRealtime opens the websocket of the application. The websocket needs a real connection,
// so the application is served by a test server for it.
func dialRealtime(t *testing.T, h *harness) *websocket.Conn {
	t.Helper()

	srv := httptest.NewServer(h.handler)
	t.Cleanup(srv.Close)

	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/api/ws", nil)
	require.NoError(t, err)
	_ = response.Body.Close()
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// readGreeting reads the greeting and the state of the dependencies a client receives first.
func readGreeting(t *testing.T, conn *websocket.Conn) {
	t.Helper()

	readData[domain.Hello](t, conn, domain.MessageHello)
	readData[map[string]string](t, conn, domain.MessageHealth)
}

// readData reads the next websocket message, checks its type, and returns its data.
func readData[T any](t *testing.T, conn *websocket.Conn, typ domain.MessageType) T {
	t.Helper()

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(eventuallyTimeout)))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)

	var message struct {
		Type domain.MessageType `json:"type"`
		Data T                  `json:"data"`
	}
	require.NoError(t, json.Unmarshal(data, &message))
	require.Equal(t, typ, message.Type)
	return message.Data
}

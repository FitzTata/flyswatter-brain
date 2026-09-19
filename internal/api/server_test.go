package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FitzTata/flyswatter-brain/internal/game"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixedController struct {
	action game.Action
}

func (c fixedController) NextAction(context.Context, game.Observation) (game.Action, error) {
	return c.action, nil
}

func TestHealth(t *testing.T) {
	t.Parallel()

	// Arrange
	server := newTestServer(t)

	// Act
	response, err := http.Get(server.URL + "/healthz")

	// Assert
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, "application/json", response.Header.Get("Content-Type"))
}

func TestWebsocketSession(t *testing.T) {
	t.Parallel()

	// Arrange
	server := newTestServer(t)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	connection, _, err := websocket.Dial(context.Background(), wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"http://localhost:5173"}},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.CloseNow() })

	// Act
	var initial serverMessage
	require.NoError(t, wsjson.Read(context.Background(), connection, &initial))
	require.NoError(t, wsjson.Write(context.Background(), connection, clientMessage{
		Type: "input",
		Input: game.Input{
			SwatterPosition: game.Vec2{X: 0.1, Y: 0.1},
		},
	}))
	var updated serverMessage
	require.NoError(t, wsjson.Read(context.Background(), connection, &updated))

	// Assert
	require.NotNil(t, initial.Snapshot)
	require.NotNil(t, updated.Snapshot)
	assert.Equal(t, "snapshot", initial.Type)
	assert.Equal(t, uint64(0), initial.Snapshot.Tick)
	assert.Equal(t, uint64(1), updated.Snapshot.Tick)
	assert.Equal(t, game.ActionStraight, updated.Snapshot.LastAction)
}

func TestWebsocketRejectsUnknownMessage(t *testing.T) {
	t.Parallel()

	// Arrange
	server := newTestServer(t)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	connection, _, err := websocket.Dial(context.Background(), wsURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.CloseNow() })
	var initial serverMessage
	require.NoError(t, wsjson.Read(context.Background(), connection, &initial))

	// Act
	require.NoError(t, wsjson.Write(context.Background(), connection, clientMessage{Type: "unknown"}))
	var response serverMessage
	require.NoError(t, wsjson.Read(context.Background(), connection, &response))

	// Assert
	assert.Equal(t, "error", response.Type)
	assert.Equal(t, "unsupported message type", response.Error)
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := NewServer(logger, func(int64) *game.Game {
		return game.New(game.DefaultConfig(), fixedController{action: game.ActionStraight})
	})
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	return httpServer
}

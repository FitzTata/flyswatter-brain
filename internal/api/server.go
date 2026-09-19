package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/FitzTata/flyswatter-brain/internal/game"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type GameFactory func(seed int64) *game.Game

type SessionStore interface {
	Load(context.Context, string) (game.Snapshot, bool, error)
	Save(context.Context, string, game.Input, game.Snapshot) error
}

type Server struct {
	logger  *slog.Logger
	factory GameFactory
	store   SessionStore
	seed    atomic.Int64
}

type clientMessage struct {
	Type  string     `json:"type"`
	Input game.Input `json:"input"`
}

type serverMessage struct {
	Type     string         `json:"type"`
	Snapshot *game.Snapshot `json:"snapshot,omitempty"`
	Error    string         `json:"error,omitempty"`
}

func NewServer(logger *slog.Logger, factory GameFactory) *Server {
	return &Server{
		logger:  logger,
		factory: factory,
	}
}

func (s *Server) WithStore(store SessionStore) *Server {
	s.store = store
	return s
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /ws", s.websocket)
	return mux
}

func (s *Server) health(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(map[string]string{"status": "ok"})
}

func (s *Server) websocket(writer http.ResponseWriter, request *http.Request) {
	session := request.URL.Query().Get("session")
	if session == "" {
		session = "default"
	}
	currentGame := s.factory(s.seed.Add(1))
	if s.store != nil {
		snapshot, found, err := s.store.Load(request.Context(), session)
		if err != nil {
			s.logger.Error("load session", "session", session, "error", err)
			http.Error(writer, "load session", http.StatusInternalServerError)
			return
		}
		if found {
			if err := currentGame.Restore(snapshot); err != nil {
				s.logger.Error("restore session", "session", session, "error", err)
				http.Error(writer, "restore session", http.StatusInternalServerError)
				return
			}
		}
	}

	connection, err := websocket.Accept(writer, request, &websocket.AcceptOptions{
		OriginPatterns: []string{"localhost:*", "127.0.0.1:*", "[::1]:*"},
	})
	if err != nil {
		s.logger.Error("accept websocket", "error", err)
		return
	}
	defer connection.CloseNow()
	connection.SetReadLimit(4 * 1024)

	if err := writeSnapshot(request.Context(), connection, currentGame.Snapshot()); err != nil {
		return
	}

	s.logger.Info("session connected", "session_id", session)

	for {
		var message clientMessage
		if err := wsjson.Read(request.Context(), connection, &message); err != nil {
			return
		}
		if message.Type != "input" {
			if err := writeError(request.Context(), connection, "unsupported message type"); err != nil {
				return
			}
			continue
		}

		snapshot, err := currentGame.Step(request.Context(), message.Input)
		if err != nil {
			if err := writeError(request.Context(), connection, err.Error()); err != nil {
				return
			}
			continue
		}
		if s.store != nil {
			if err := s.store.Save(request.Context(), session, message.Input, snapshot); err != nil {
				s.logger.Error("save session", "session", session, "error", err)
				_ = writeError(request.Context(), connection, "save session")
				return
			}
		}
		if err := writeSnapshot(request.Context(), connection, snapshot); err != nil {
			return
		}
	}
}

func writeSnapshot(ctx context.Context, connection *websocket.Conn, snapshot game.Snapshot) error {
	return wsjson.Write(ctx, connection, serverMessage{
		Type:     "snapshot",
		Snapshot: &snapshot,
	})
}

func writeError(ctx context.Context, connection *websocket.Conn, message string) error {
	return wsjson.Write(ctx, connection, serverMessage{
		Type:  "error",
		Error: message,
	})
}

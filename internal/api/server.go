package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/FitzTata/flyswatter-brain/internal/game"
	"github.com/FitzTata/flyswatter-brain/internal/neural"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type GameFactory func(seed int64, mode neural.Mode) (*game.Game, error)

type SessionStore interface {
	Load(context.Context, string) (game.Snapshot, bool, error)
	Save(context.Context, string, game.Input, game.Snapshot) error
}

type Server struct {
	logger  *slog.Logger
	factory GameFactory
	store   SessionStore
	seed    atomic.Int64
	players sync.Map
}

type playerSession struct {
	conn *websocket.Conn
	gen  uint64
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

var playerIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

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
	query := request.URL.Query()
	playerID := strings.TrimSpace(query.Get("session"))
	if playerID == "" {
		playerID = strings.TrimSpace(query.Get("player"))
	}
	if playerID == "" {
		http.Error(writer, "player required", http.StatusBadRequest)
		return
	}
	playerID = strings.ToLower(playerID)
	if !playerIDPattern.MatchString(playerID) {
		http.Error(writer, "invalid player", http.StatusBadRequest)
		return
	}
	mode, err := neural.ParseMode(query.Get("mode"))
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	currentGame, err := s.factory(s.seed.Add(1), mode)
	if err != nil {
		s.logger.Error("create game", "player", playerID, "mode", mode, "error", err)
		http.Error(writer, "create game", http.StatusInternalServerError)
		return
	}
	if s.store != nil {
		snapshot, found, err := s.store.Load(request.Context(), playerID)
		if err != nil {
			s.logger.Error("load session", "session", playerID, "error", err)
			http.Error(writer, "load session", http.StatusInternalServerError)
			return
		}
		if found {
			if err := currentGame.Restore(snapshot); err != nil {
				s.logger.Error("restore session", "session", playerID, "error", err)
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
	connection.SetReadLimit(4 * 1024)

	gen := uint64(s.seed.Add(1))
	if previous, loaded := s.players.Swap(playerID, &playerSession{conn: connection, gen: gen}); loaded {
		if old, ok := previous.(*playerSession); ok && old.conn != nil {
			_ = old.conn.Close(websocket.StatusPolicyViolation, "session_replaced")
		}
	}
	defer func() {
		if current, ok := s.players.Load(playerID); ok {
			if entry, ok := current.(*playerSession); ok && entry.gen == gen {
				s.players.Delete(playerID)
			}
		}
		_ = connection.CloseNow()
	}()

	if err := writeSnapshot(request.Context(), connection, currentGame.Snapshot()); err != nil {
		return
	}

	s.logger.Info("session connected", "session_id", playerID, "mode", mode)

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
			if err := s.store.Save(request.Context(), playerID, message.Input, snapshot); err != nil {
				s.logger.Error("save session", "session", playerID, "error", err)
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

package persistence

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/FitzTata/flyswatter-brain/internal/game"
)

const checkpointVersion = 1

var (
	ErrInvalidSession = errors.New("invalid session id")
	sessionPattern    = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
)

type FileStore struct {
	root string
	mu   sync.Mutex
}

type checkpoint struct {
	Version  int           `json:"version"`
	SavedAt  time.Time     `json:"saved_at"`
	Snapshot game.Snapshot `json:"snapshot"`
}

type event struct {
	SavedAt time.Time   `json:"saved_at"`
	Tick    uint64      `json:"tick"`
	Episode uint64      `json:"episode"`
	Input   game.Input  `json:"input"`
	Action  game.Action `json:"action"`
	Alive   bool        `json:"alive"`
}

func NewFileStore(root string) *FileStore {
	return &FileStore{root: root}
}

func (s *FileStore) Load(ctx context.Context, session string) (game.Snapshot, bool, error) {
	if !sessionPattern.MatchString(session) {
		return game.Snapshot{}, false, ErrInvalidSession
	}
	if err := ctx.Err(); err != nil {
		return game.Snapshot{}, false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.checkpointPath(session))
	if errors.Is(err, os.ErrNotExist) {
		return game.Snapshot{}, false, nil
	}
	if err != nil {
		return game.Snapshot{}, false, fmt.Errorf("read checkpoint: %w", err)
	}

	var saved checkpoint
	if err := json.Unmarshal(data, &saved); err != nil {
		return game.Snapshot{}, false, fmt.Errorf("decode checkpoint: %w", err)
	}
	if saved.Version != checkpointVersion {
		return game.Snapshot{}, false, fmt.Errorf("checkpoint version %d: %w", saved.Version, game.ErrInvalidCheckpoint)
	}
	return saved.Snapshot, true, nil
}

func (s *FileStore) Save(ctx context.Context, session string, input game.Input, snapshot game.Snapshot) error {
	if !sessionPattern.MatchString(session) {
		return ErrInvalidSession
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	directory := s.sessionDirectory(session)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create session directory: %w", err)
	}

	savedAt := time.Now().UTC()
	if err := appendJSONLine(s.eventsPath(session), event{
		SavedAt: savedAt,
		Tick:    snapshot.Tick,
		Episode: snapshot.Episode,
		Input:   input,
		Action:  snapshot.LastAction,
		Alive:   snapshot.Alive,
	}); err != nil {
		return fmt.Errorf("append event: %w", err)
	}

	snapshot.NeuralActivity = nil
	data, err := json.MarshalIndent(checkpoint{
		Version:  checkpointVersion,
		SavedAt:  savedAt,
		Snapshot: snapshot,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode checkpoint: %w", err)
	}
	if err := writeAtomic(s.checkpointPath(session), append(data, '\n')); err != nil {
		return fmt.Errorf("write checkpoint: %w", err)
	}
	return nil
}

func (s *FileStore) sessionDirectory(session string) string {
	return filepath.Join(s.root, session)
}

func (s *FileStore) checkpointPath(session string) string {
	return filepath.Join(s.sessionDirectory(session), "checkpoint.json")
}

func (s *FileStore) eventsPath(session string) string {
	return filepath.Join(s.sessionDirectory(session), "events.jsonl")
}

func appendJSONLine(path string, value any) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(file)
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		_ = file.Close()
		return err
	}
	if err := writer.Flush(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func writeAtomic(path string, data []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), "checkpoint-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	_ = temporary.Chmod(0o600)
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err == nil {
		return nil
	}
	// Windows cannot rename over an existing destination.
	_ = os.Remove(path)
	return os.Rename(temporaryPath, path)
}


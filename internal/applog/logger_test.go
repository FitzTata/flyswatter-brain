package applog

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJSONLogger(t *testing.T) {
	t.Parallel()

	logger, closer, err := New(Config{Level: "info", Format: "json"})

	require.NoError(t, err)
	require.Nil(t, closer)
	require.NotNil(t, logger)
	assert.True(t, logger.Enabled(t.Context(), slog.LevelInfo))
}

func TestNewRejectsInvalidLevel(t *testing.T) {
	t.Parallel()

	_, _, err := New(Config{Level: "verbose"})

	require.Error(t, err)
}

func TestNewWritesOptionalFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "app.log")
	logger, closer, err := New(Config{Level: "debug", Format: "text", File: path})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, closer.Close())
	})

	logger.Info("hello", "session_id", "default")

	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(contents), "hello")
	assert.Contains(t, string(contents), "session_id")
}

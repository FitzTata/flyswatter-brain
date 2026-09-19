package persistence

import (
	"bufio"
	"context"
	"os"
	"testing"

	"github.com/FitzTata/flyswatter-brain/internal/game"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStoreRoundTrip(t *testing.T) {
	t.Parallel()

	store := NewFileStore(t.TempDir())
	input := game.Input{SwatterPosition: game.Vec2{X: 0.2, Y: 0.3}}
	first := checkpointSnapshot(1)
	latest := checkpointSnapshot(2)
	latest.NeuralActivity = &game.NeuralActivity{Model: "transient"}

	require.NoError(t, store.Save(context.Background(), "demo", input, first))
	require.NoError(t, store.Save(context.Background(), "demo", input, latest))
	restored, found, err := store.Load(context.Background(), "demo")

	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, latest.Tick, restored.Tick)
	assert.Nil(t, restored.NeuralActivity)
	assert.Equal(t, 2, lineCount(t, store.eventsPath("demo")))
}

func TestFileStoreMissingCheckpoint(t *testing.T) {
	t.Parallel()

	store := NewFileStore(t.TempDir())

	_, found, err := store.Load(context.Background(), "missing")

	require.NoError(t, err)
	assert.False(t, found)
}

func TestFileStoreRejectsInvalidSession(t *testing.T) {
	t.Parallel()

	store := NewFileStore(t.TempDir())

	_, _, err := store.Load(context.Background(), "../escape")

	require.ErrorIs(t, err, ErrInvalidSession)
}

func checkpointSnapshot(tick uint64) game.Snapshot {
	return game.Snapshot{
		Tick:       tick,
		Episode:    1,
		Alive:      true,
		SurvivalMS: int64(tick * 20),
		LastAction: game.ActionStraight,
		Fly: game.Fly{
			Position: game.Vec2{X: 0.5, Y: 0.5},
			Speed:    0.28,
			Radius:   0.025,
		},
		Swatter: game.Swatter{
			Position: game.Vec2{X: 0.8, Y: 0.2},
			Radius:   0.09,
		},
	}
}

func lineCount(t *testing.T, path string) int {
	t.Helper()

	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		count++
	}
	require.NoError(t, scanner.Err())
	return count
}

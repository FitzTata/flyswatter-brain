package appconfig

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	t.Parallel()

	config, err := load(t.TempDir(), lookup(nil))

	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:8080", config.Address)
	assert.Equal(t, "auto", config.ControllerMode)
	assert.Equal(t, 20*time.Millisecond, time.Duration(config.Game.StepSeconds*float64(time.Second)))
	assert.Equal(t, 0.28, config.Game.FlySpeed)
	assert.Equal(t, 2.0, config.NeuralStepMS)
	assert.Equal(t, 2*time.Minute, config.NeuralStartupTimeout)
}

func TestLoadOverrides(t *testing.T) {
	t.Parallel()

	config, err := load(t.TempDir(), lookup(map[string]string{
		"FLYSWATTER_ADDR":                   "0.0.0.0:9000",
		"FLYSWATTER_CONTROLLER":             "random",
		"FLYSWATTER_GAME_STEP_MS":           "25",
		"FLYSWATTER_FLY_SPEED":              "0.45",
		"FLYSWATTER_FLY_RADIUS":             "0.03",
		"FLYSWATTER_SWATTER_RADIUS":         "0.08",
		"FLYSWATTER_TURN_DEGREES":           "30",
		"FLYSWATTER_ESCAPE_MULTIPLIER":      "3",
		"FLYSWATTER_NEURAL_STEP_MS":         "4",
		"FLYSWATTER_NEURAL_STARTUP_TIMEOUT": "30s",
	}))

	require.NoError(t, err)
	assert.Equal(t, "0.0.0.0:9000", config.Address)
	assert.Equal(t, "random", config.ControllerMode)
	assert.InDelta(t, 0.025, config.Game.StepSeconds, 0.0001)
	assert.Equal(t, 0.45, config.Game.FlySpeed)
	assert.InDelta(t, math.Pi/6, config.Game.TurnRadians, 0.0001)
	assert.Equal(t, 3.0, config.Game.EscapeMultiplier)
	assert.Equal(t, 4.0, config.NeuralStepMS)
	assert.Equal(t, 30*time.Second, config.NeuralStartupTimeout)
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values map[string]string
	}{
		{name: "controller", values: map[string]string{"FLYSWATTER_CONTROLLER": "unknown"}},
		{name: "speed", values: map[string]string{"FLYSWATTER_FLY_SPEED": "0"}},
		{name: "duration", values: map[string]string{"FLYSWATTER_NEURAL_STARTUP_TIMEOUT": "soon"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := load(t.TempDir(), lookup(tt.values))

			require.Error(t, err)
		})
	}
}

func lookup(values map[string]string) func(string) string {
	return func(name string) string {
		return values[name]
	}
}

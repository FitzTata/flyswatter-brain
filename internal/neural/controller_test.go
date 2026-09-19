package neural

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/FitzTata/flyswatter-brain/internal/game"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestController(t *testing.T) {
	t.Parallel()

	controller, err := NewController(context.Background(), Config{
		Python:         os.Args[0],
		WorkerScript:   "-test.run=TestWorkerProcess",
		StartupTimeout: time.Second,
		StepDurationMS: 50,
		ExtraEnv:       []string{"GO_WANT_NEURAL_HELPER=1"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, controller.Close()) })

	action, err := controller.NextAction(context.Background(), game.Observation{})

	require.NoError(t, err)
	assert.Equal(t, game.ActionTurnRight, action)
	activity := controller.LastNeuralActivity()
	require.NotNil(t, activity)
	assert.Equal(t, "MaleCNS v1.0", activity.Model)
	assert.Equal(t, 14.0, activity.RightHz)
}

func TestControllerConfigureAndSave(t *testing.T) {
	t.Parallel()

	controller, err := NewController(context.Background(), Config{
		Python:         os.Args[0],
		WorkerScript:   "-test.run=TestWorkerProcess",
		StartupTimeout: time.Second,
		StepDurationMS: 50,
		ExtraEnv:       []string{"GO_WANT_NEURAL_HELPER=1"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, controller.Close()) })

	require.NoError(t, controller.Configure(true, "", false))
	require.NoError(t, controller.SaveCheckpoint(t.TempDir()+"/brain.npz"))
}

func TestWorkerProcess(t *testing.T) {
	if os.Getenv("GO_WANT_NEURAL_HELPER") != "1" {
		return
	}

	encoder := json.NewEncoder(os.Stdout)
	require.NoError(t, encoder.Encode(workerMessage{Type: "ready"}))
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var request map[string]any
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &request))
		switch request["type"] {
		case "configure":
			require.NoError(t, encoder.Encode(workerMessage{Type: "configured"}))
		case "save":
			require.NoError(t, encoder.Encode(workerMessage{Type: "saved"}))
		default:
			require.NoError(t, encoder.Encode(workerMessage{
				Type:   "decision",
				Action: game.ActionTurnRight,
				NeuralActivity: &game.NeuralActivity{
					Model:   "MaleCNS v1.0",
					RightHz: 14,
				},
			}))
		}
	}
}

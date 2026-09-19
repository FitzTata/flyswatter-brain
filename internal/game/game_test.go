package game

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixedController struct {
	action Action
	err    error
}

func (c fixedController) NextAction(context.Context, Observation) (Action, error) {
	return c.action, c.err
}

type telemetryController struct {
	fixedController
	activity NeuralActivity
}

func (c telemetryController) LastNeuralActivity() *NeuralActivity {
	return &c.activity
}

func TestGameStep(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		action        Action
		wantHeading   float64
		wantSpeed     float64
		wantMinTravel float64
	}{
		{name: "straight", action: ActionStraight, wantHeading: 0, wantSpeed: 0.18, wantMinTravel: 0.008},
		{name: "left", action: ActionTurnLeft, wantHeading: -math.Pi / 12, wantSpeed: 0.18, wantMinTravel: 0.008},
		{name: "right", action: ActionTurnRight, wantHeading: math.Pi / 12, wantSpeed: 0.18, wantMinTravel: 0.008},
		{name: "escape", action: ActionEscape, wantHeading: 0, wantSpeed: 0.432, wantMinTravel: 0.02},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			instance := New(DefaultConfig(), fixedController{action: tt.action})

			// Act
			snapshot, err := instance.Step(context.Background(), Input{})

			// Assert
			require.NoError(t, err)
			assert.Equal(t, uint64(1), snapshot.Tick)
			assert.Equal(t, tt.action, snapshot.LastAction)
			assert.InDelta(t, tt.wantHeading, snapshot.Fly.Heading, 0.0001)
			assert.InDelta(t, tt.wantSpeed, snapshot.Fly.Speed, 0.0001)
			assert.Greater(t, math.Hypot(snapshot.Fly.Position.X-0.5, snapshot.Fly.Position.Y-0.5), tt.wantMinTravel)
		})
	}
}

func TestGameCollisionStartsNewEpisodeOnNextStep(t *testing.T) {
	t.Parallel()

	// Arrange
	config := DefaultConfig()
	instance := New(config, fixedController{action: ActionStraight})
	hitPosition := Vec2{X: 0.5 + config.FlySpeed*config.StepSeconds, Y: 0.5}

	// Act
	hit, err := instance.Step(context.Background(), Input{
		SwatterPosition: hitPosition,
		Attacking:       true,
	})
	require.NoError(t, err)
	held, err := instance.Step(context.Background(), Input{
		SwatterPosition: hitPosition,
		Attacking:       true,
	})
	require.NoError(t, err)
	next, err := instance.Step(context.Background(), Input{
		SwatterPosition: Vec2{},
	})

	// Assert
	require.NoError(t, err)
	assert.False(t, hit.Alive)
	assert.Equal(t, uint64(1), hit.Episode)
	assert.True(t, held.Alive)
	assert.False(t, held.Swatter.Attacking)
	assert.Equal(t, uint64(2), held.Episode)
	assert.Equal(t, int64(50), held.SurvivalMS)
	assert.True(t, next.Alive)
	assert.Equal(t, uint64(2), next.Episode)
	assert.Equal(t, int64(100), next.SurvivalMS)
}

func TestGameRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	// Arrange
	instance := New(DefaultConfig(), fixedController{action: ActionStraight})

	// Act
	_, err := instance.Step(context.Background(), Input{
		SwatterPosition: Vec2{X: math.NaN()},
	})

	// Assert
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestRandomControllerIsDeterministic(t *testing.T) {
	t.Parallel()

	// Arrange
	first := NewRandomController(42)
	second := NewRandomController(42)

	// Act
	var firstActions, secondActions []Action
	for range 20 {
		action, err := first.NextAction(context.Background(), Observation{})
		require.NoError(t, err)
		firstActions = append(firstActions, action)

		action, err = second.NextAction(context.Background(), Observation{})
		require.NoError(t, err)
		secondActions = append(secondActions, action)
	}

	// Assert
	assert.Equal(t, firstActions, secondActions)
}

func TestGameIncludesNeuralActivity(t *testing.T) {
	t.Parallel()

	instance := New(DefaultConfig(), telemetryController{
		fixedController: fixedController{action: ActionStraight},
		activity:        NeuralActivity{Model: "MaleCNS v1.0", RightHz: 14},
	})

	snapshot, err := instance.Step(context.Background(), Input{})

	require.NoError(t, err)
	require.NotNil(t, snapshot.NeuralActivity)
	assert.Equal(t, "MaleCNS v1.0", snapshot.NeuralActivity.Model)
	assert.Equal(t, 14.0, snapshot.NeuralActivity.RightHz)
}

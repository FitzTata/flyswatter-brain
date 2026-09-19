package game

import (
	"context"
	"math"
	"math/rand"
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

func newTestGame(config Config, controller Controller) *Game {
	return NewWithRand(config, controller, rand.New(rand.NewSource(1)))
}

func TestGameStep(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		action          Action
		wantHeadingDelta float64
		wantSpeed       float64
		wantMinTravel   float64
	}{
		{name: "straight", action: ActionStraight, wantHeadingDelta: 0, wantSpeed: 0.42, wantMinTravel: 0.008},
		{name: "left", action: ActionTurnLeft, wantHeadingDelta: -math.Pi / 12, wantSpeed: 0.42, wantMinTravel: 0.008},
		{name: "right", action: ActionTurnRight, wantHeadingDelta: math.Pi / 12, wantSpeed: 0.42, wantMinTravel: 0.008},
		{name: "escape", action: ActionEscape, wantHeadingDelta: 0, wantSpeed: 1.47, wantMinTravel: 0.028},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			instance := newTestGame(DefaultConfig(), fixedController{action: tt.action})
			start := instance.Snapshot().Fly

			snapshot, err := instance.Step(context.Background(), Input{})

			require.NoError(t, err)
			assert.Equal(t, uint64(1), snapshot.Tick)
			assert.Equal(t, tt.action, snapshot.LastAction)
			assert.InDelta(t, start.Heading+tt.wantHeadingDelta, snapshot.Fly.Heading, 0.0001)
			assert.InDelta(t, tt.wantSpeed, snapshot.Fly.Speed, 0.0001)
			assert.Greater(
				t,
				math.Hypot(snapshot.Fly.Position.X-start.Position.X, snapshot.Fly.Position.Y-start.Position.Y),
				tt.wantMinTravel,
			)
		})
	}
}

func TestGameCollisionStartsNewEpisodeOnNextStep(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	config.SwingWindupSeconds = config.StepSeconds
	config.SwingActiveSeconds = config.StepSeconds
	instance := newTestGame(config, fixedController{action: ActionStraight})

	windup, err := instance.Step(context.Background(), Input{
		SwatterPosition: instance.Snapshot().Fly.Position,
		Attacking:       true,
	})
	require.NoError(t, err)
	hit, err := instance.Step(context.Background(), Input{
		SwatterPosition: instance.Snapshot().Fly.Position,
		Attacking:       true,
	})
	require.NoError(t, err)
	held, err := instance.Step(context.Background(), Input{
		SwatterPosition: instance.Snapshot().Fly.Position,
		Attacking:       true,
	})
	require.NoError(t, err)
	next, err := instance.Step(context.Background(), Input{
		SwatterPosition: Vec2{},
	})

	require.NoError(t, err)
	assert.True(t, windup.Alive)
	assert.Equal(t, SwingWindup, windup.Swatter.Phase)
	assert.False(t, hit.Alive)
	assert.Equal(t, uint64(1), hit.Episode)
	assert.True(t, held.Alive)
	assert.False(t, held.Swatter.Attacking)
	assert.Equal(t, uint64(2), held.Episode)
	assert.Equal(t, int64(20), held.SurvivalMS)
	assert.True(t, next.Alive)
	assert.Equal(t, uint64(2), next.Episode)
	assert.Equal(t, int64(40), next.SurvivalMS)
}

func TestSwingWindupIsNotLethal(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	config.SwingWindupSeconds = 0.1
	config.SwingActiveSeconds = 0.02
	instance := newTestGame(config, fixedController{action: ActionStraight})
	fly := instance.Snapshot().Fly.Position

	windup, err := instance.Step(context.Background(), Input{
		SwatterPosition: fly,
		Attacking:       true,
	})

	require.NoError(t, err)
	assert.True(t, windup.Alive)
	assert.True(t, windup.Swatter.Attacking)
	assert.Equal(t, SwingWindup, windup.Swatter.Phase)
}

func TestFlySpawnsAtRandomPositions(t *testing.T) {
	t.Parallel()

	first := newTestGame(DefaultConfig(), fixedController{action: ActionStraight}).Snapshot().Fly
	second := NewWithRand(DefaultConfig(), fixedController{action: ActionStraight}, rand.New(rand.NewSource(2))).Snapshot().Fly

	assert.NotEqual(t, first.Position, second.Position)
	assert.NotEqual(t, first.Heading, second.Heading)
	assert.GreaterOrEqual(t, first.Position.X, DefaultConfig().FlyRadius)
	assert.LessOrEqual(t, first.Position.X, 1-DefaultConfig().FlyRadius)
}

func TestSwatterHitboxMatchesRenderedEllipse(t *testing.T) {
	t.Parallel()

	const aspectRatio = 2.0
	swatter := Swatter{
		Position:  Vec2{X: 0.5, Y: 0.5},
		Radius:    0.09,
		Attacking: true,
	}
	tests := []struct {
		name     string
		position Vec2
		want     bool
	}{
		{name: "center", position: swatter.Position, want: true},
		{name: "inside long axis", position: ellipsePoint(swatter.Position, 0, 0.11, aspectRatio), want: true},
		{name: "outside short axis", position: ellipsePoint(swatter.Position, 0.1, 0, aspectRatio), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fly := Fly{Position: tt.position, Radius: 0.025}

			assert.Equal(t, tt.want, collides(fly, swatter, aspectRatio))
		})
	}
}

func TestGameRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	instance := newTestGame(DefaultConfig(), fixedController{action: ActionStraight})

	_, err := instance.Step(context.Background(), Input{
		SwatterPosition: Vec2{X: math.NaN()},
	})

	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestGameRestoresCheckpointWithoutTransientState(t *testing.T) {
	t.Parallel()

	instance := newTestGame(DefaultConfig(), fixedController{action: ActionStraight})
	checkpoint := Snapshot{
		Tick:       42,
		Episode:    3,
		Alive:      true,
		SurvivalMS: 840,
		LastAction: ActionTurnLeft,
		Fly:        Fly{Position: Vec2{X: 0.3, Y: 0.4}, Heading: 1, Speed: 0.42, Radius: 0.025},
		Swatter:    Swatter{Position: Vec2{X: 0.8, Y: 0.2}, Radius: 0.085, Attacking: true},
		NeuralActivity: &NeuralActivity{
			Model: "stale",
		},
	}

	err := instance.Restore(checkpoint)
	restored := instance.Snapshot()

	require.NoError(t, err)
	assert.Equal(t, uint64(42), restored.Tick)
	assert.Equal(t, uint64(3), restored.Episode)
	assert.Equal(t, checkpoint.Fly, restored.Fly)
	assert.False(t, restored.Swatter.Attacking)
	assert.Nil(t, restored.NeuralActivity)
}

func TestGameRejectsInvalidCheckpoint(t *testing.T) {
	t.Parallel()

	instance := newTestGame(DefaultConfig(), fixedController{action: ActionStraight})

	err := instance.Restore(Snapshot{})

	require.ErrorIs(t, err, ErrInvalidCheckpoint)
}

func TestRandomControllerIsDeterministic(t *testing.T) {
	t.Parallel()

	first := NewRandomController(42)
	second := NewRandomController(42)

	var firstActions, secondActions []Action
	for range 20 {
		action, err := first.NextAction(context.Background(), Observation{})
		require.NoError(t, err)
		firstActions = append(firstActions, action)

		action, err = second.NextAction(context.Background(), Observation{})
		require.NoError(t, err)
		secondActions = append(secondActions, action)
	}

	assert.Equal(t, firstActions, secondActions)
}

func TestGameIncludesNeuralActivity(t *testing.T) {
	t.Parallel()

	instance := newTestGame(DefaultConfig(), telemetryController{
		fixedController: fixedController{action: ActionStraight},
		activity:        NeuralActivity{Model: "MaleCNS v1.0", RightHz: 14},
	})

	snapshot, err := instance.Step(context.Background(), Input{})

	require.NoError(t, err)
	require.NotNil(t, snapshot.NeuralActivity)
	assert.Equal(t, "MaleCNS v1.0", snapshot.NeuralActivity.Model)
	assert.Equal(t, 14.0, snapshot.NeuralActivity.RightHz)
}

func ellipsePoint(center Vec2, localX, localY, aspectRatio float64) Vec2 {
	const angle = -math.Pi / 4
	dx := localX*math.Cos(angle) - localY*math.Sin(angle)
	dy := localX*math.Sin(angle) + localY*math.Cos(angle)
	return Vec2{X: center.X + dx/aspectRatio, Y: center.Y + dy}
}

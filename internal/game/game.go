package game

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

type Action string

const (
	ActionTurnLeft  Action = "turn_left"
	ActionTurnRight Action = "turn_right"
	ActionStraight  Action = "straight"
	ActionEscape    Action = "escape"
)

var (
	ErrInvalidAction     = errors.New("invalid controller action")
	ErrInvalidInput      = errors.New("invalid game input")
	ErrInvalidCheckpoint = errors.New("invalid game checkpoint")
)

type Vec2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Fly struct {
	Position Vec2    `json:"position"`
	Heading  float64 `json:"heading"`
	Speed    float64 `json:"speed"`
	Radius   float64 `json:"radius"`
}

type SwingPhase string

const (
	SwingIdle   SwingPhase = ""
	SwingWindup SwingPhase = "windup"
	SwingStrike SwingPhase = "strike"
)

type Swatter struct {
	Position  Vec2       `json:"position"`
	Radius    float64    `json:"radius"`
	Attacking bool       `json:"attacking"`
	Phase     SwingPhase `json:"phase,omitempty"`
}

type Input struct {
	SwatterPosition  Vec2    `json:"swatter_position"`
	Attacking        bool    `json:"attacking"`
	ArenaAspectRatio float64 `json:"arena_aspect_ratio,omitempty"`
}

type Observation struct {
	Tick    uint64  `json:"tick"`
	Fly     Fly     `json:"fly"`
	Swatter Swatter `json:"swatter"`
}

type Snapshot struct {
	Tick           uint64          `json:"tick"`
	Episode        uint64          `json:"episode"`
	Alive          bool            `json:"alive"`
	SurvivalMS     int64           `json:"survival_ms"`
	LastAction     Action          `json:"last_action"`
	Fly            Fly             `json:"fly"`
	Swatter        Swatter         `json:"swatter"`
	NeuralActivity *NeuralActivity `json:"neural_activity,omitempty"`
}

type NeuralActivity struct {
	Model        string       `json:"model"`
	ModelTimeMS  float64      `json:"model_time_ms"`
	LeftHz       float64      `json:"left_hz"`
	RightHz      float64      `json:"right_hz"`
	DifferenceHz float64      `json:"difference_hz"`
	GateSpikes   int          `json:"gate_spikes"`
	TotalSpikes  int          `json:"total_spikes"`
	StepSeconds  float64      `json:"step_seconds"`
	Nodes        []NeuralNode `json:"nodes"`
}

type NeuralNode struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	Spikes int     `json:"spikes"`
	RateHz float64 `json:"rate_hz"`
}

type Controller interface {
	NextAction(context.Context, Observation) (Action, error)
}

type NeuralActivityProvider interface {
	LastNeuralActivity() *NeuralActivity
}

type Config struct {
	StepSeconds        float64
	FlySpeed           float64
	FlyRadius          float64
	SwatterRadius      float64
	TurnRadians        float64
	EscapeMultiplier   float64
	SwingWindupSeconds float64
	SwingActiveSeconds float64
}

func DefaultConfig() Config {
	return Config{
		StepSeconds:        0.02,
		FlySpeed:           0.42,
		FlyRadius:          0.025,
		SwatterRadius:      0.085,
		TurnRadians:        math.Pi / 12,
		EscapeMultiplier:   3.5,
		SwingWindupSeconds: 0.28,
		SwingActiveSeconds: 0.08,
	}
}

type Game struct {
	config           Config
	controller       Controller
	random           *rand.Rand
	state            Snapshot
	attackHeld       bool
	swingRemaining   float64
	swingPhase       SwingPhase
	arenaAspectRatio float64
}

func New(config Config, controller Controller) *Game {
	return NewWithRand(config, controller, rand.New(rand.NewSource(time.Now().UnixNano())))
}

func NewWithRand(config Config, controller Controller, random *rand.Rand) *Game {
	game := &Game{
		config:           config,
		controller:       controller,
		random:           random,
		arenaAspectRatio: 2,
		state: Snapshot{
			Episode: 1,
			Alive:   true,
		},
	}
	game.resetFly()
	return game
}

func (g *Game) Snapshot() Snapshot {
	return g.state
}

func (g *Game) ReportDeath() {
	if reporter, ok := g.controller.(interface{ ReportDeath() }); ok {
		reporter.ReportDeath()
	}
}

func (g *Game) Restore(snapshot Snapshot) error {
	if snapshot.Episode == 0 ||
		snapshot.SurvivalMS < 0 ||
		!validAction(snapshot.LastAction) ||
		!validEntity(snapshot.Fly.Position, snapshot.Fly.Radius, snapshot.Fly.Speed) ||
		!validEntity(snapshot.Swatter.Position, snapshot.Swatter.Radius, 0) ||
		!isFinite(snapshot.Fly.Heading) {
		return ErrInvalidCheckpoint
	}

	snapshot.Swatter.Attacking = false
	snapshot.Swatter.Phase = SwingIdle
	snapshot.NeuralActivity = nil
	g.state = snapshot
	g.attackHeld = false
	g.swingPhase = SwingIdle
	g.swingRemaining = 0
	return nil
}

func (g *Game) Step(ctx context.Context, input Input) (Snapshot, error) {
	if !validPosition(input.SwatterPosition) || !validAspectRatio(input.ArenaAspectRatio) {
		return Snapshot{}, ErrInvalidInput
	}
	if input.ArenaAspectRatio > 0 {
		g.arenaAspectRatio = input.ArenaAspectRatio
	}
	strike := input.Attacking && !g.attackHeld
	g.attackHeld = input.Attacking
	startedSwing := false
	if strike && g.swingPhase == SwingIdle {
		g.swingPhase = SwingWindup
		g.swingRemaining = g.config.SwingWindupSeconds
		startedSwing = true
	}
	if !startedSwing {
		g.advanceSwing()
	}
	swatter := Swatter{
		Position: Vec2{
			X: clamp(input.SwatterPosition.X, 0, 1),
			Y: clamp(input.SwatterPosition.Y, 0, 1),
		},
		Radius:    g.config.SwatterRadius,
		Attacking: g.swingPhase != SwingIdle,
		Phase:     g.swingPhase,
	}
	if !g.state.Alive {
		g.state.Episode++
		g.state.Alive = true
		g.state.SurvivalMS = 0
		g.resetFly()
	}

	g.state.Swatter = swatter
	if collides(g.state.Fly, g.state.Swatter, g.arenaAspectRatio) {
		g.state.Tick++
		g.state.Alive = false
		g.swingPhase = SwingIdle
		g.swingRemaining = 0
		g.state.Swatter.Attacking = false
		g.state.Swatter.Phase = SwingIdle
		return g.state, nil
	}

	action, err := g.controller.NextAction(ctx, Observation{
		Tick:    g.state.Tick,
		Fly:     g.state.Fly,
		Swatter: g.state.Swatter,
	})
	if err != nil {
		return Snapshot{}, err
	}
	if !validAction(action) {
		return Snapshot{}, ErrInvalidAction
	}
	g.state.NeuralActivity = nil
	if provider, ok := g.controller.(NeuralActivityProvider); ok {
		g.state.NeuralActivity = provider.LastNeuralActivity()
	}

	g.apply(action)
	g.state.Tick++
	g.state.SurvivalMS += int64(math.Round(g.config.StepSeconds * 1000))
	g.state.LastAction = action
	g.state.Alive = !collides(g.state.Fly, g.state.Swatter, g.arenaAspectRatio)

	return g.state, nil
}

func (g *Game) advanceSwing() {
	if g.swingPhase == SwingIdle {
		return
	}
	g.swingRemaining -= g.config.StepSeconds
	if g.swingRemaining > 0 {
		return
	}
	switch g.swingPhase {
	case SwingWindup:
		g.swingPhase = SwingStrike
		g.swingRemaining = g.config.SwingActiveSeconds
	default:
		g.swingPhase = SwingIdle
		g.swingRemaining = 0
	}
}

func (g *Game) apply(action Action) {
	switch action {
	case ActionTurnLeft:
		g.state.Fly.Heading -= g.config.TurnRadians
	case ActionTurnRight:
		g.state.Fly.Heading += g.config.TurnRadians
	}

	speed := g.config.FlySpeed
	if action == ActionEscape {
		speed *= g.config.EscapeMultiplier
	}
	g.state.Fly.Speed = speed

	distance := speed * g.config.StepSeconds
	g.state.Fly.Position.X += math.Cos(g.state.Fly.Heading) * distance
	g.state.Fly.Position.Y += math.Sin(g.state.Fly.Heading) * distance
	g.reflectAtBounds()
}

func (g *Game) reflectAtBounds() {
	radius := g.state.Fly.Radius
	if g.state.Fly.Position.X < radius || g.state.Fly.Position.X > 1-radius {
		g.state.Fly.Position.X = clamp(g.state.Fly.Position.X, radius, 1-radius)
		g.state.Fly.Heading = math.Pi - g.state.Fly.Heading
	}
	if g.state.Fly.Position.Y < radius || g.state.Fly.Position.Y > 1-radius {
		g.state.Fly.Position.Y = clamp(g.state.Fly.Position.Y, radius, 1-radius)
		g.state.Fly.Heading = -g.state.Fly.Heading
	}
}

func (g *Game) resetFly() {
	margin := g.config.FlyRadius + 0.05
	g.state.Fly = Fly{
		Position: Vec2{
			X: margin + g.random.Float64()*(1-2*margin),
			Y: margin + g.random.Float64()*(1-2*margin),
		},
		Heading: g.random.Float64() * 2 * math.Pi,
		Speed:   g.config.FlySpeed,
		Radius:  g.config.FlyRadius,
	}
	g.state.Swatter = Swatter{
		Position: Vec2{X: 0.5, Y: 0.8},
		Radius:   g.config.SwatterRadius,
	}
	g.state.LastAction = ActionStraight
}

func collides(fly Fly, swatter Swatter, aspectRatio float64) bool {
	if !swatter.Attacking || swatter.Phase == SwingWindup {
		return false
	}

	dx := (fly.Position.X - swatter.Position.X) * aspectRatio
	dy := fly.Position.Y - swatter.Position.Y
	cos, sin := math.Cos(-math.Pi/4), math.Sin(-math.Pi/4)
	localX := dx*cos + dy*sin
	localY := -dx*sin + dy*cos
	radiusX := swatter.Radius*0.78 + fly.Radius
	radiusY := swatter.Radius + fly.Radius

	return math.Pow(localX/radiusX, 2)+math.Pow(localY/radiusY, 2) <= 1
}

func validAction(action Action) bool {
	switch action {
	case ActionTurnLeft, ActionTurnRight, ActionStraight, ActionEscape:
		return true
	default:
		return false
	}
}

func validPosition(position Vec2) bool {
	return isFinite(position.X) && isFinite(position.Y)
}

func validAspectRatio(value float64) bool {
	return value >= 0 && isFinite(value)
}

func validEntity(position Vec2, radius, speed float64) bool {
	return validPosition(position) &&
		position.X >= 0 && position.X <= 1 &&
		position.Y >= 0 && position.Y <= 1 &&
		radius > 0 && isFinite(radius) &&
		speed >= 0 && isFinite(speed)
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func clamp(value, minValue, maxValue float64) float64 {
	return math.Max(minValue, math.Min(maxValue, value))
}

package neural

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/FitzTata/flyswatter-brain/internal/game"
)

type Mode string

const ModeShared Mode = "shared"

const rewardEveryTicks = 50
const escapeBiasTicks = 25 // ~0.5s at 20ms steps

type Hub struct {
	mu               sync.Mutex
	controller       *Controller
	sharedPath       string
	mode             Mode
	learning         bool
	pendingAverse    bool
	aliveTicks       int
	escapeBiasLeft   int
	available        bool
}

func NewHub(controller *Controller, runsDir string) *Hub {
	return &Hub{
		controller: controller,
		sharedPath: filepath.Join(runsDir, "_shared", "brain.npz"),
		mode:       ModeShared,
		available:  controller != nil,
	}
}

func (h *Hub) Available() bool {
	return h != nil && h.available
}

func (h *Hub) Close() error {
	if h == nil || h.controller == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	_ = h.controller.SaveCheckpoint(h.sharedPath)
	return h.controller.Close()
}

func (h *Hub) ControllerFor(mode Mode, seed int64) (game.Controller, error) {
	if mode != ModeShared {
		return nil, fmt.Errorf("invalid mode %q", mode)
	}
	if !h.Available() {
		return game.NewRandomController(seed), nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.applySharedLocked(); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *Hub) applySharedLocked() error {
	if h.controller == nil {
		return errors.New("neural worker unavailable")
	}
	if err := h.controller.Configure(true, h.sharedPath, false); err != nil {
		return err
	}
	h.mode = ModeShared
	h.learning = true
	h.pendingAverse = false
	h.aliveTicks = 0
	h.escapeBiasLeft = 0
	return nil
}

func (h *Hub) NextAction(ctx context.Context, observation game.Observation) (game.Action, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	reinforcement := "none"
	if h.learning {
		if h.pendingAverse {
			reinforcement = "aversive"
			h.pendingAverse = false
			h.aliveTicks = 0
		} else {
			h.aliveTicks++
			if h.aliveTicks%rewardEveryTicks == 0 {
				reinforcement = "reward"
			}
		}
	}
	action, err := h.controller.NextActionWith(ctx, observation, reinforcement)
	if err != nil {
		return "", err
	}
	action, h.escapeBiasLeft = biasEscape(action, h.escapeBiasLeft)
	return action, nil
}

func (h *Hub) LastNeuralActivity() *game.NeuralActivity {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.controller == nil {
		return nil
	}
	return h.controller.LastNeuralActivity()
}

func (h *Hub) ReportDeath() {
	h.ReportHit()
}

func (h *Hub) ReportHit() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.learning {
		return
	}
	h.pendingAverse = true
	h.aliveTicks = 0
	h.escapeBiasLeft = escapeBiasTicks
}

func biasEscape(action game.Action, ticksLeft int) (game.Action, int) {
	if ticksLeft <= 0 {
		return action, 0
	}
	ticksLeft--
	if action == game.ActionStraight {
		return game.ActionEscape, ticksLeft
	}
	return action, ticksLeft
}

func ParseMode(raw string) (Mode, error) {
	switch Mode(raw) {
	case "", ModeShared:
		return ModeShared, nil
	default:
		return "", fmt.Errorf("mode must be shared")
	}
}

func (h *Hub) Mode() Mode {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.mode
}

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

const (
	ModeShared Mode = "shared"
	ModeStatic Mode = "static"
	ModeRandom Mode = "random"
)

const rewardEveryTicks = 50

type Hub struct {
	mu            sync.Mutex
	controller    *Controller
	sharedPath    string
	mode          Mode
	learning      bool
	pendingAverse bool
	aliveTicks    int
	available     bool
}

func NewHub(controller *Controller, runsDir string) *Hub {
	return &Hub{
		controller: controller,
		sharedPath: filepath.Join(runsDir, "_shared", "brain.npz"),
		mode:       ModeStatic,
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
	if h.mode == ModeShared {
		_ = h.controller.SaveCheckpoint(h.sharedPath)
	}
	return h.controller.Close()
}

func (h *Hub) ControllerFor(mode Mode, seed int64) (game.Controller, error) {
	if mode == ModeRandom || !h.Available() {
		return game.NewRandomController(seed), nil
	}
	if mode != ModeShared && mode != ModeStatic {
		return nil, fmt.Errorf("invalid mode %q", mode)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.applyModeLocked(mode); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *Hub) applyModeLocked(mode Mode) error {
	if h.controller == nil {
		return errors.New("neural worker unavailable")
	}
	if h.mode == ModeShared && mode != ModeShared {
		if err := h.controller.SaveCheckpoint(h.sharedPath); err != nil {
			return err
		}
	}
	learning := mode == ModeShared
	loadPath := ""
	if mode == ModeShared {
		loadPath = h.sharedPath
	}
	if err := h.controller.Configure(learning, loadPath, mode == ModeStatic); err != nil {
		return err
	}
	h.mode = mode
	h.learning = learning
	h.pendingAverse = false
	h.aliveTicks = 0
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
	return h.controller.NextActionWith(ctx, observation, reinforcement)
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
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.learning {
		h.pendingAverse = true
		h.aliveTicks = 0
	}
}

func ParseMode(raw string) (Mode, error) {
	switch Mode(raw) {
	case "", ModeStatic:
		return ModeStatic, nil
	case ModeShared, ModeRandom:
		return Mode(raw), nil
	default:
		return "", fmt.Errorf("mode must be shared, static, or random")
	}
}

func (h *Hub) Mode() Mode {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.mode
}

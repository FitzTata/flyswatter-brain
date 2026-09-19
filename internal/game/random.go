package game

import (
	"context"
	"math/rand"
	"sync"
)

type RandomController struct {
	mu     sync.Mutex
	random *rand.Rand
}

func NewRandomController(seed int64) *RandomController {
	return &RandomController{
		random: rand.New(rand.NewSource(seed)),
	}
}

func (c *RandomController) NextAction(context.Context, Observation) (Action, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch c.random.Intn(10) {
	case 0, 1, 2:
		return ActionTurnLeft, nil
	case 3, 4, 5:
		return ActionTurnRight, nil
	case 6, 7, 8:
		return ActionStraight, nil
	default:
		return ActionEscape, nil
	}
}

package neural

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/FitzTata/flyswatter-brain/internal/game"
)

type Config struct {
	Python         string
	WorkerScript   string
	StonkflySource string
	DataDir        string
	StartupTimeout time.Duration
	StepDurationMS float64
	ExtraEnv       []string
}

type Controller struct {
	mu       sync.Mutex
	command  *exec.Cmd
	stdin    io.WriteCloser
	scanner  *bufio.Scanner
	activity *game.NeuralActivity
	stepMS   float64
}

type workerMessage struct {
	Type           string               `json:"type"`
	Model          string               `json:"model,omitempty"`
	Neurons        int                  `json:"neurons,omitempty"`
	Connections    int                  `json:"connections,omitempty"`
	Action         game.Action          `json:"action,omitempty"`
	NeuralActivity *game.NeuralActivity `json:"neural_activity,omitempty"`
	Error          string               `json:"error,omitempty"`
}

func NewController(ctx context.Context, config Config) (*Controller, error) {
	command := exec.Command(config.Python, config.WorkerScript)
	command.Env = append(
		os.Environ(),
		"FLYSWATTER_STONKFLY_SOURCE="+config.StonkflySource,
		"STONKFLY_DATA="+config.DataDir,
	)
	command.Env = append(command.Env, config.ExtraEnv...)
	command.Stderr = os.Stderr

	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("worker stdin: %w", err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("worker stdout: %w", err)
	}
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("start worker: %w", err)
	}

	controller := &Controller{
		command: command,
		stdin:   stdin,
		scanner: bufio.NewScanner(stdout),
		stepMS:  config.StepDurationMS,
	}
	controller.scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	ready := make(chan error, 1)
	go func() {
		message, err := controller.read()
		if err == nil && message.Type != "ready" {
			err = fmt.Errorf("unexpected startup message: %s", message.Type)
		}
		ready <- err
	}()

	timeout := config.StartupTimeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	select {
	case err := <-ready:
		if err != nil {
			_ = command.Process.Kill()
			_ = command.Wait()
			return nil, fmt.Errorf("initialize worker: %w", err)
		}
	case <-ctx.Done():
		_ = command.Process.Kill()
		_ = command.Wait()
		return nil, ctx.Err()
	case <-time.After(timeout):
		_ = command.Process.Kill()
		_ = command.Wait()
		return nil, errors.New("initialize worker: timeout")
	}

	return controller, nil
}

func (c *Controller) NextAction(_ context.Context, observation game.Observation) (game.Action, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	request := struct {
		Type        string           `json:"type"`
		Observation game.Observation `json:"observation"`
		DurationMS  float64          `json:"duration_ms"`
	}{
		Type:        "step",
		Observation: observation,
		DurationMS:  c.stepMS,
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	if _, err := c.stdin.Write(append(payload, '\n')); err != nil {
		return "", fmt.Errorf("write worker request: %w", err)
	}

	message, err := c.read()
	if err != nil {
		return "", err
	}
	if message.Type == "error" {
		return "", errors.New(message.Error)
	}
	if message.Type != "decision" {
		return "", fmt.Errorf("unexpected worker message: %s", message.Type)
	}
	c.activity = message.NeuralActivity
	return message.Action, nil
}

func (c *Controller) LastNeuralActivity() *game.NeuralActivity {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activity == nil {
		return nil
	}
	activity := *c.activity
	activity.Nodes = append([]game.NeuralNode(nil), c.activity.Nodes...)
	return &activity
}

func (c *Controller) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_ = c.stdin.Close()
	done := make(chan error, 1)
	go func() {
		done <- c.command.Wait()
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		_ = c.command.Process.Kill()
		return <-done
	}
}

func (c *Controller) read() (workerMessage, error) {
	if !c.scanner.Scan() {
		if err := c.scanner.Err(); err != nil {
			return workerMessage{}, fmt.Errorf("read worker response: %w", err)
		}
		return workerMessage{}, errors.New("worker exited")
	}
	var message workerMessage
	if err := json.Unmarshal(c.scanner.Bytes(), &message); err != nil {
		return workerMessage{}, fmt.Errorf("decode worker response: %w", err)
	}
	return message, nil
}

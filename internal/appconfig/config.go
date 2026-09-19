package appconfig

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/FitzTata/flyswatter-brain/internal/game"
)

type Config struct {
	Address              string
	RunsDir              string
	ControllerMode       string
	Python               string
	WorkerScript         string
	StonkflySource       string
	DataDir              string
	NeuralStartupTimeout time.Duration
	NeuralStepMS         float64
	Game                 game.Config
}

func Load() (Config, error) {
	root, err := os.Getwd()
	if err != nil {
		return Config{}, fmt.Errorf("working directory: %w", err)
	}
	return load(root, os.Getenv)
}

func load(root string, getenv func(string) string) (Config, error) {
	gameConfig := game.DefaultConfig()
	gameStepMS, err := positiveFloat(getenv, "FLYSWATTER_GAME_STEP_MS", gameConfig.StepSeconds*1000)
	if err != nil {
		return Config{}, err
	}
	gameConfig.StepSeconds = gameStepMS / 1000
	if gameConfig.FlySpeed, err = positiveFloat(getenv, "FLYSWATTER_FLY_SPEED", gameConfig.FlySpeed); err != nil {
		return Config{}, err
	}
	if gameConfig.FlyRadius, err = positiveFloat(getenv, "FLYSWATTER_FLY_RADIUS", gameConfig.FlyRadius); err != nil {
		return Config{}, err
	}
	if gameConfig.SwatterRadius, err = positiveFloat(getenv, "FLYSWATTER_SWATTER_RADIUS", gameConfig.SwatterRadius); err != nil {
		return Config{}, err
	}
	turnDegrees, err := positiveFloat(getenv, "FLYSWATTER_TURN_DEGREES", gameConfig.TurnRadians*180/math.Pi)
	if err != nil {
		return Config{}, err
	}
	gameConfig.TurnRadians = turnDegrees * math.Pi / 180
	if gameConfig.EscapeMultiplier, err = positiveFloat(
		getenv,
		"FLYSWATTER_ESCAPE_MULTIPLIER",
		gameConfig.EscapeMultiplier,
	); err != nil {
		return Config{}, err
	}
	swingWindupMS, err := positiveFloat(
		getenv,
		"FLYSWATTER_SWING_WINDUP_MS",
		gameConfig.SwingWindupSeconds*1000,
	)
	if err != nil {
		return Config{}, err
	}
	gameConfig.SwingWindupSeconds = swingWindupMS / 1000
	swingActiveMS, err := positiveFloat(
		getenv,
		"FLYSWATTER_SWING_ACTIVE_MS",
		gameConfig.SwingActiveSeconds*1000,
	)
	if err != nil {
		return Config{}, err
	}
	gameConfig.SwingActiveSeconds = swingActiveMS / 1000

	controllerMode := valueOr(getenv("FLYSWATTER_CONTROLLER"), "auto")
	if controllerMode != "auto" && controllerMode != "random" && controllerMode != "malecns" {
		return Config{}, errors.New("FLYSWATTER_CONTROLLER must be auto, random, or malecns")
	}
	neuralStepMS, err := positiveFloat(getenv, "FLYSWATTER_NEURAL_STEP_MS", 2)
	if err != nil {
		return Config{}, err
	}
	startupTimeout, err := positiveDuration(getenv, "FLYSWATTER_NEURAL_STARTUP_TIMEOUT", 2*time.Minute)
	if err != nil {
		return Config{}, err
	}

	python := filepath.Join(root, ".venv-neural", "bin", "python")
	if runtime.GOOS == "windows" {
		python = filepath.Join(root, ".venv-neural", "Scripts", "python.exe")
	}
	return Config{
		Address:              valueOr(getenv("FLYSWATTER_ADDR"), "127.0.0.1:8080"),
		RunsDir:              valueOr(getenv("FLYSWATTER_RUNS_DIR"), filepath.Join(root, "runs")),
		ControllerMode:       controllerMode,
		Python:               valueOr(getenv("FLYSWATTER_PYTHON"), python),
		WorkerScript:         filepath.Join(root, "neural_worker", "worker.py"),
		StonkflySource:       filepath.Join(root, "third_party", "stonkfly"),
		DataDir:              valueOr(getenv("STONKFLY_DATA"), filepath.Join(root, ".local", "malecns")),
		NeuralStartupTimeout: startupTimeout,
		NeuralStepMS:         neuralStepMS,
		Game:                 gameConfig,
	}, nil
}

func positiveFloat(getenv func(string) string, name string, fallback float64) (float64, error) {
	raw := getenv(name)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("%s must be a positive number", name)
	}
	return value, nil
}

func positiveDuration(getenv func(string) string, name string, fallback time.Duration) (time.Duration, error) {
	raw := getenv(name)
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", name)
	}
	return value, nil
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

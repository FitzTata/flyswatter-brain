package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/FitzTata/flyswatter-brain/internal/api"
	"github.com/FitzTata/flyswatter-brain/internal/game"
	"github.com/FitzTata/flyswatter-brain/internal/neural"
	"github.com/FitzTata/flyswatter-brain/internal/persistence"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	neuralController := connectNeuralController(ctx, logger)
	if neuralController != nil {
		defer func() {
			if err := neuralController.Close(); err != nil {
				logger.Error("close neural worker", "error", err)
			}
		}()
	}
	server := api.NewServer(logger, func(seed int64) *game.Game {
		if neuralController != nil {
			return game.New(game.DefaultConfig(), neuralController)
		}
		return game.New(game.DefaultConfig(), game.NewRandomController(seed))
	}).WithStore(persistence.NewFileStore(envOr("FLYSWATTER_RUNS_DIR", "runs")))

	httpServer := &http.Server{
		Addr:              address(),
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown server", "error", err)
		}
	}()

	logger.Info("server listening", "address", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("serve", "error", err)
		os.Exit(1)
	}
}

func connectNeuralController(ctx context.Context, logger *slog.Logger) *neural.Controller {
	mode := os.Getenv("FLYSWATTER_CONTROLLER")
	if mode == "" {
		mode = "auto"
	}
	if mode == "random" {
		logger.Info("using random controller")
		return nil
	}

	root, err := os.Getwd()
	if err != nil {
		logger.Error("resolve working directory", "error", err)
		return nil
	}
	python := filepath.Join(root, ".venv-neural", "bin", "python")
	if runtime.GOOS == "windows" {
		python = filepath.Join(root, ".venv-neural", "Scripts", "python.exe")
	}
	controller, err := neural.NewController(ctx, neural.Config{
		Python:         envOr("FLYSWATTER_PYTHON", python),
		WorkerScript:   filepath.Join(root, "neural_worker", "worker.py"),
		StonkflySource: filepath.Join(root, "third_party", "stonkfly"),
		DataDir:        envOr("STONKFLY_DATA", filepath.Join(root, ".local", "malecns")),
		StartupTimeout: 2 * time.Minute,
		StepDurationMS: 2,
	})
	if err == nil {
		logger.Info("using MaleCNS controller")
		return controller
	}
	if mode == "malecns" {
		logger.Error("MaleCNS controller required but unavailable", "error", err)
		os.Exit(1)
	}
	logger.Warn("MaleCNS unavailable; using random controller", "error", err)
	return nil
}

func address() string {
	return envOr("FLYSWATTER_ADDR", "127.0.0.1:8080")
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

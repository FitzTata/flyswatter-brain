package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FitzTata/flyswatter-brain/internal/api"
	"github.com/FitzTata/flyswatter-brain/internal/appconfig"
	"github.com/FitzTata/flyswatter-brain/internal/game"
	"github.com/FitzTata/flyswatter-brain/internal/neural"
	"github.com/FitzTata/flyswatter-brain/internal/persistence"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	config, err := appconfig.Load()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}
	neuralController := connectNeuralController(ctx, logger, config)
	if neuralController != nil {
		defer func() {
			if err := neuralController.Close(); err != nil {
				logger.Error("close neural worker", "error", err)
			}
		}()
	}
	server := api.NewServer(logger, func(seed int64) *game.Game {
		if neuralController != nil {
			return game.New(config.Game, neuralController)
		}
		return game.New(config.Game, game.NewRandomController(seed))
	}).WithStore(persistence.NewFileStore(config.RunsDir))

	httpServer := &http.Server{
		Addr:              config.Address,
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

func connectNeuralController(ctx context.Context, logger *slog.Logger, config appconfig.Config) *neural.Controller {
	if config.ControllerMode == "random" {
		logger.Info("using random controller")
		return nil
	}

	controller, err := neural.NewController(ctx, neural.Config{
		Python:         config.Python,
		WorkerScript:   config.WorkerScript,
		StonkflySource: config.StonkflySource,
		DataDir:        config.DataDir,
		StartupTimeout: config.NeuralStartupTimeout,
		StepDurationMS: config.NeuralStepMS,
	})
	if err == nil {
		logger.Info("using MaleCNS controller")
		return controller
	}
	if config.ControllerMode == "malecns" {
		logger.Error("MaleCNS controller required but unavailable", "error", err)
		os.Exit(1)
	}
	logger.Warn("MaleCNS unavailable; using random controller", "error", err)
	return nil
}

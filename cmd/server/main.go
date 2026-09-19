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
	"github.com/FitzTata/flyswatter-brain/internal/applog"
	"github.com/FitzTata/flyswatter-brain/internal/game"
	"github.com/FitzTata/flyswatter-brain/internal/neural"
	"github.com/FitzTata/flyswatter-brain/internal/persistence"
)

func main() {
	config, err := appconfig.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}
	logger, logCloser, err := applog.New(applog.Config{
		Level:  config.LogLevel,
		Format: config.LogFormat,
		File:   config.LogFile,
	})
	if err != nil {
		slog.Error("configure logger", "error", err)
		os.Exit(1)
	}
	if logCloser != nil {
		defer func() {
			_ = logCloser.Close()
		}()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	hub := connectNeuralHub(ctx, logger, config)
	if hub != nil {
		defer func() {
			if err := hub.Close(); err != nil {
				logger.Error("close neural hub", "error", err)
			}
		}()
	}

	server := api.NewServer(logger, func(seed int64, mode neural.Mode) (*game.Game, error) {
		if mode == neural.ModeRandom || hub == nil || !hub.Available() {
			if mode == neural.ModeShared || mode == neural.ModeStatic {
				if config.ControllerMode == "malecns" {
					return nil, errors.New("MaleCNS controller unavailable")
				}
				logger.Warn("MaleCNS unavailable; falling back to random", "requested_mode", mode)
			}
			return game.New(config.Game, game.NewRandomController(seed)), nil
		}
		controller, err := hub.ControllerFor(mode, seed)
		if err != nil {
			return nil, err
		}
		return game.New(config.Game, controller), nil
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

func connectNeuralHub(ctx context.Context, logger *slog.Logger, config appconfig.Config) *neural.Hub {
	if config.ControllerMode == "random" {
		logger.Info("using random controller only")
		return neural.NewHub(nil, config.RunsDir)
	}

	controller, err := neural.NewController(ctx, neural.Config{
		Python:         config.Python,
		WorkerScript:   config.WorkerScript,
		StonkflySource: config.StonkflySource,
		DataDir:        config.DataDir,
		StartupTimeout: config.NeuralStartupTimeout,
		StepDurationMS: config.NeuralStepMS,
		ExtraEnv: []string{
			"FLYSWATTER_LOG_LEVEL=" + config.LogLevel,
		},
	})
	if err == nil {
		logger.Info("using MaleCNS neural hub")
		return neural.NewHub(controller, config.RunsDir)
	}
	if config.ControllerMode == "malecns" {
		logger.Error("MaleCNS controller required but unavailable", "error", err)
		os.Exit(1)
	}
	logger.Warn("MaleCNS unavailable; shared/static fall back to random", "error", err)
	return neural.NewHub(nil, config.RunsDir)
}

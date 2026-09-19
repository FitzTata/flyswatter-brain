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
	"github.com/FitzTata/flyswatter-brain/internal/game"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	server := api.NewServer(logger, func(seed int64) *game.Game {
		return game.New(game.DefaultConfig(), game.NewRandomController(seed))
	})

	httpServer := &http.Server{
		Addr:              address(),
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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

func address() string {
	if value := os.Getenv("FLYSWATTER_ADDR"); value != "" {
		return value
	}
	return "127.0.0.1:8080"
}

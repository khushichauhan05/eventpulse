package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/apekshita/eventpulse/internal/config"
	"github.com/apekshita/eventpulse/internal/database"
	"github.com/apekshita/eventpulse/internal/health"
	"github.com/apekshita/eventpulse/internal/logging"
	_ "github.com/apekshita/eventpulse/internal/metrics"
	"github.com/apekshita/eventpulse/internal/services"
)

func main() {
	cfg := config.Load("alert-service")
	logger := logging.New(cfg.ServiceName, cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.OpenPostgres(ctx, cfg.DatabaseDSN)
	if err != nil {
		logger.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.Handle("/health", health.NewHandler(cfg.ServiceName))
	mux.Handle("/metrics", promhttp.Handler())

	go func() {
		if err := services.StartAlertHealthServer(ctx, cfg, logger, mux); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("alert health server stopped", "error", err)
		}
	}()

	if err := services.RunAlert(ctx, cfg, db, logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("alert service stopped", "error", err)
		os.Exit(1)
	}
}

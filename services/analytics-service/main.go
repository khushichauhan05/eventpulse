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
	"github.com/apekshita/eventpulse/internal/health"
	"github.com/apekshita/eventpulse/internal/logging"
	_ "github.com/apekshita/eventpulse/internal/metrics"
	"github.com/apekshita/eventpulse/internal/services"
)

func main() {
	cfg := config.Load("analytics-service")
	logger := logging.New(cfg.ServiceName, cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	mux.Handle("/health", health.NewHandler(cfg.ServiceName))
	mux.Handle("/metrics", promhttp.Handler())

	go func() {
		if err := services.StartAnalyticsHealthServer(ctx, cfg, logger, mux); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("analytics health server stopped", "error", err)
		}
	}()

	if err := services.RunAnalytics(ctx, cfg, logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("analytics service stopped", "error", err)
		os.Exit(1)
	}
}

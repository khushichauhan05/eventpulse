package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/apekshita/eventpulse/internal/config"
	"github.com/apekshita/eventpulse/internal/database"
	"github.com/apekshita/eventpulse/internal/handlers"
	"github.com/apekshita/eventpulse/internal/kafka"
	"github.com/apekshita/eventpulse/internal/logging"
)

func main() {
	cfg := config.Load("api-gateway")
	logger := logging.New(cfg.ServiceName, cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.OpenPostgres(ctx, cfg.DatabaseDSN)
	if err != nil {
		logger.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	writer := kafka.NewWriter(cfg.KafkaBrokers, cfg.RawTopic)
	defer writer.Close()

	h := &handlers.GatewayHandler{
		Logger: logger,
		DB:     db,
		Writer: writer,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/events", h.CreateEvent)
	mux.HandleFunc("/alerts", h.GetAlerts)
	mux.HandleFunc("/alert", h.GetAlertByID)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	logger.Info("api gateway started", "port", cfg.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("api gateway stopped", "error", err)
		os.Exit(1)
	}
}

package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/apekshita/eventpulse/internal/config"
	"github.com/apekshita/eventpulse/internal/kafka"
	"github.com/apekshita/eventpulse/internal/models"
	"github.com/apekshita/eventpulse/internal/retry"
)

func RunAnalytics(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	reader := kafka.NewReader(cfg.KafkaBrokers, cfg.RawTopic, cfg.AnalyticsGroup)
	defer reader.Close()

	writer := kafka.NewWriter(cfg.KafkaBrokers, cfg.ProcessedTopic)
	defer writer.Close()

	logger.Info("analytics service started")

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			logger.Error("failed to fetch kafka message", "error", err)
			continue
		}

		var event models.Event
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			logger.Warn("invalid event payload", "error", err, "topic", cfg.RawTopic)
			_ = retry.Do(ctx, 3, 200*time.Millisecond, func() error {
				return reader.CommitMessages(ctx, msg)
			})
			continue
		}

		riskScore := 20
		if event.Amount > 10000 {
			riskScore = 90
		}

		processed := models.ProcessedEvent{
			UserID:    event.UserID,
			EventType: event.EventType,
			Amount:    event.Amount,
			RiskScore: riskScore,
		}

		payload, err := json.Marshal(processed)
		if err != nil {
			logger.Error("failed to marshal processed event", "error", err)
			continue
		}

		if err := kafka.WriteWithRetry(ctx, writer, payload); err != nil {
			logger.Error("failed to publish processed event", "error", err)
			continue
		}

		if err := retry.Do(ctx, 5, 200*time.Millisecond, func() error {
			return reader.CommitMessages(ctx, msg)
		}); err != nil {
			logger.Error("failed to commit analytics offset", "error", err)
			continue
		}

		logger.Info("processed event", "user_id", processed.UserID, "risk_score", processed.RiskScore)
	}
}

func StartAnalyticsHealthServer(ctx context.Context, cfg config.Config, logger *slog.Logger, handler http.Handler) error {
	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.HealthPort),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	logger.Info("analytics health server started", "port", cfg.HealthPort)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

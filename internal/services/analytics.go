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
	"github.com/apekshita/eventpulse/internal/metrics"
	"github.com/apekshita/eventpulse/internal/models"
	"github.com/apekshita/eventpulse/internal/retry"
)

const analyticsService = "analytics-service"

// ScoreEvent returns a risk score for the given transaction amount.
// Amounts above 10 000 are classified as high risk (score 90).
func ScoreEvent(amount float64) int {
	if amount > 10000 {
		return 90
	}
	return 20
}

func RunAnalytics(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	reader := kafka.NewReader(cfg.KafkaBrokers, cfg.RawTopic, cfg.AnalyticsGroup)
	defer reader.Close()

	writer := kafka.NewWriter(cfg.KafkaBrokers, cfg.ProcessedTopic)
	defer writer.Close()

	dlq := kafka.NewWriter(cfg.KafkaBrokers, cfg.DLQTopic)
	defer dlq.Close()

	logger.Info("analytics service started")

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			logger.Error("failed to fetch kafka message", "error", err)
			metrics.ProcessingErrors.WithLabelValues(analyticsService, "kafka_read").Inc()
			continue
		}

		metrics.EventsConsumed.WithLabelValues(analyticsService, cfg.RawTopic).Inc()

		var event models.Event
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			logger.Warn("malformed event — routing to DLQ", "error", err, "topic", cfg.RawTopic)
			_ = kafka.WriteWithRetry(ctx, dlq, msg.Value)
			metrics.DLQMessages.WithLabelValues(analyticsService, "unmarshal_error").Inc()
			_ = retry.Do(ctx, 3, 200*time.Millisecond, func() error {
				return reader.CommitMessages(ctx, msg)
			})
			continue
		}

		riskScore := ScoreEvent(event.Amount)
		metrics.RiskScores.Observe(float64(riskScore))

		processed := models.ProcessedEvent{
			EventID:   event.EventID,
			UserID:    event.UserID,
			EventType: event.EventType,
			Amount:    event.Amount,
			RiskScore: riskScore,
		}

		payload, err := json.Marshal(processed)
		if err != nil {
			logger.Error("failed to marshal processed event", "error", err)
			metrics.ProcessingErrors.WithLabelValues(analyticsService, "marshal").Inc()
			continue
		}

		if err := kafka.WriteWithRetry(ctx, writer, payload); err != nil {
			logger.Error("failed to publish processed event", "error", err)
			metrics.ProcessingErrors.WithLabelValues(analyticsService, "kafka_publish").Inc()
			continue
		}

		if err := retry.Do(ctx, 5, 200*time.Millisecond, func() error {
			return reader.CommitMessages(ctx, msg)
		}); err != nil {
			logger.Error("failed to commit analytics offset", "error", err)
			continue
		}

		metrics.EventsPublished.WithLabelValues(analyticsService, cfg.ProcessedTopic).Inc()
		metrics.EventsProcessed.WithLabelValues(analyticsService).Inc()
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

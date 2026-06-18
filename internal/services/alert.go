package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/apekshita/eventpulse/internal/config"
	internalKafka "github.com/apekshita/eventpulse/internal/kafka"
	"github.com/apekshita/eventpulse/internal/metrics"
	"github.com/apekshita/eventpulse/internal/models"
	"github.com/apekshita/eventpulse/internal/retry"
)

const alertService = "alert-service"

// IsHighRisk reports whether an event with the given score should trigger an alert.
func IsHighRisk(riskScore int) bool {
	return riskScore >= 80
}

func RunAlert(ctx context.Context, cfg config.Config, db *sql.DB, logger *slog.Logger) error {
	reader := internalKafka.NewReader(cfg.KafkaBrokers, cfg.ProcessedTopic, cfg.AlertGroup)
	defer reader.Close()

	writer := internalKafka.NewWriter(cfg.KafkaBrokers, cfg.AlertsTopic)
	defer writer.Close()

	dlq := internalKafka.NewWriter(cfg.KafkaBrokers, cfg.DLQTopic)
	defer dlq.Close()

	logger.Info("alert service started")

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			logger.Error("failed to fetch kafka message", "error", err)
			metrics.ProcessingErrors.WithLabelValues(alertService, "kafka_read").Inc()
			continue
		}

		metrics.EventsConsumed.WithLabelValues(alertService, cfg.ProcessedTopic).Inc()

		var event models.ProcessedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			logger.Warn("malformed processed event — routing to DLQ", "error", err, "topic", cfg.ProcessedTopic)
			_ = internalKafka.WriteWithRetry(ctx, dlq, msg.Value)
			metrics.DLQMessages.WithLabelValues(alertService, "unmarshal_error").Inc()
			_ = retry.Do(ctx, 3, 200*time.Millisecond, func() error {
				return reader.CommitMessages(ctx, msg)
			})
			continue
		}

		if !IsHighRisk(event.RiskScore) {
			_ = retry.Do(ctx, 3, 200*time.Millisecond, func() error {
				return reader.CommitMessages(ctx, msg)
			})
			continue
		}

		alert := models.Alert{
			EventID:   event.EventID,
			UserID:    event.UserID,
			RiskScore: event.RiskScore,
			Message:   "HIGH RISK TRANSACTION DETECTED",
		}

		payload, err := json.Marshal(alert)
		if err != nil {
			logger.Error("failed to marshal alert", "error", err)
			metrics.ProcessingErrors.WithLabelValues(alertService, "marshal").Inc()
			continue
		}

		if err := internalKafka.WriteWithRetry(ctx, writer, payload); err != nil {
			logger.Error("failed to publish alert", "error", err)
			metrics.ProcessingErrors.WithLabelValues(alertService, "kafka_publish").Inc()
			continue
		}

		// ON CONFLICT DO NOTHING provides idempotency — duplicate event_ids are silently skipped.
		if err := retry.Do(ctx, 5, 200*time.Millisecond, func() error {
			_, err := db.ExecContext(ctx,
				`INSERT INTO alerts (event_id, user_id, risk_score, message)
				 VALUES ($1, $2, $3, $4)
				 ON CONFLICT (event_id) DO NOTHING`,
				alert.EventID, alert.UserID, alert.RiskScore, alert.Message)
			return err
		}); err != nil {
			logger.Error("failed to store alert", "error", err)
			metrics.ProcessingErrors.WithLabelValues(alertService, "db_insert").Inc()
			continue
		}

		if err := retry.Do(ctx, 5, 200*time.Millisecond, func() error {
			return reader.CommitMessages(ctx, msg)
		}); err != nil {
			logger.Error("failed to commit alert offset", "error", err)
			continue
		}

		metrics.AlertsGenerated.Inc()
		metrics.EventsProcessed.WithLabelValues(alertService).Inc()
		logger.Info("alert generated", "user_id", alert.UserID, "risk_score", alert.RiskScore, "event_id", alert.EventID)
	}
}

func StartAlertHealthServer(ctx context.Context, cfg config.Config, logger *slog.Logger, handler http.Handler) error {
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

	logger.Info("alert health server started", "port", cfg.HealthPort)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

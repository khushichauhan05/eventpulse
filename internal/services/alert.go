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
	"github.com/apekshita/eventpulse/internal/models"
	"github.com/apekshita/eventpulse/internal/retry"
)

func RunAlert(ctx context.Context, cfg config.Config, db *sql.DB, logger *slog.Logger) error {
	reader := internalKafka.NewReader(cfg.KafkaBrokers, cfg.ProcessedTopic, cfg.AlertGroup)
	defer reader.Close()

	writer := internalKafka.NewWriter(cfg.KafkaBrokers, cfg.AlertsTopic)
	defer writer.Close()

	logger.Info("alert service started")

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			logger.Error("failed to fetch kafka message", "error", err)
			continue
		}

		var event models.ProcessedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			logger.Warn("invalid processed event payload", "error", err, "topic", cfg.ProcessedTopic)
			_ = retry.Do(ctx, 3, 200*time.Millisecond, func() error {
				return reader.CommitMessages(ctx, msg)
			})
			continue
		}

		if event.RiskScore < 80 {
			if err := retry.Do(ctx, 3, 200*time.Millisecond, func() error {
				return reader.CommitMessages(ctx, msg)
			}); err != nil {
				logger.Error("failed to commit low-risk offset", "error", err)
			}
			continue
		}

		alert := models.Alert{
			UserID:    event.UserID,
			RiskScore: event.RiskScore,
			Message:   "HIGH RISK TRANSACTION DETECTED",
		}

		payload, err := json.Marshal(alert)
		if err != nil {
			logger.Error("failed to marshal alert", "error", err)
			continue
		}

		if err := internalKafka.WriteWithRetry(ctx, writer, payload); err != nil {
			logger.Error("failed to publish alert", "error", err)
			continue
		}

		if err := retry.Do(ctx, 5, 200*time.Millisecond, func() error {
			_, err := db.ExecContext(ctx, `INSERT INTO alerts (user_id, risk_score, message) VALUES ($1, $2, $3)`, alert.UserID, alert.RiskScore, alert.Message)
			return err
		}); err != nil {
			logger.Error("failed to store alert", "error", err)
			continue
		}

		if err := retry.Do(ctx, 5, 200*time.Millisecond, func() error {
			return reader.CommitMessages(ctx, msg)
		}); err != nil {
			logger.Error("failed to commit alert offset", "error", err)
			continue
		}

		logger.Info("alert generated", "user_id", alert.UserID, "risk_score", alert.RiskScore)
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

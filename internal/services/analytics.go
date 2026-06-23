package services

import (
	"bytes"
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

var mlHTTPClient = &http.Client{Timeout: 2 * time.Second}

// ScoreEvent is the rule-based fallback used when the ML service is unavailable.
func ScoreEvent(amount float64) int {
	if amount > 10000 {
		return 90
	}
	return 20
}

// scoreWithML calls the Python ML service and returns its response.
// Returns an error if the service is unreachable or returns a non-200 status.
func scoreWithML(ctx context.Context, mlURL string, event models.Event) (models.MLScoreResponse, error) {
	body, err := json.Marshal(models.MLScoreRequest{
		UserID:    event.UserID,
		Amount:    event.Amount,
		EventType: event.EventType,
	})
	if err != nil {
		return models.MLScoreResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mlURL+"/score", bytes.NewReader(body))
	if err != nil {
		return models.MLScoreResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := mlHTTPClient.Do(req)
	if err != nil {
		return models.MLScoreResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.MLScoreResponse{}, fmt.Errorf("ml service returned HTTP %d", resp.StatusCode)
	}

	var result models.MLScoreResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return models.MLScoreResponse{}, err
	}
	return result, nil
}

func RunAnalytics(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	reader := kafka.NewReader(cfg.KafkaBrokers, cfg.RawTopic, cfg.AnalyticsGroup)
	defer reader.Close()

	writer := kafka.NewWriter(cfg.KafkaBrokers, cfg.ProcessedTopic)
	defer writer.Close()

	dlq := kafka.NewWriter(cfg.KafkaBrokers, cfg.DLQTopic)
	defer dlq.Close()

	logger.Info("analytics service started", "ml_service_url", cfg.MLServiceURL)

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
			logger.Warn("malformed event — routing to DLQ", "error", err)
			_ = kafka.WriteWithRetry(ctx, dlq, msg.Value)
			metrics.DLQMessages.WithLabelValues(analyticsService, "unmarshal_error").Inc()
			_ = retry.Do(ctx, 3, 200*time.Millisecond, func() error {
				return reader.CommitMessages(ctx, msg)
			})
			continue
		}

		// ── ML scoring with rule-based fallback ──────────────────────────────
		var riskScore int
		var confidence float64
		var explanation map[string]float64
		var mlScored bool

		if cfg.MLServiceURL != "" {
			t0 := time.Now()
			mlResp, mlErr := scoreWithML(ctx, cfg.MLServiceURL, event)
			elapsed := time.Since(t0)

			if mlErr != nil {
				logger.Warn("ml service unavailable, falling back to rule-based scoring",
					"error", mlErr, "user_id", event.UserID)
				metrics.MLFallbacks.Inc()
				riskScore = ScoreEvent(event.Amount)
			} else {
				riskScore = mlResp.RiskScore
				confidence = mlResp.Confidence
				explanation = mlResp.Explanation
				mlScored = true
				metrics.MLScoredEvents.Inc()
				metrics.MLInferenceLatency.Observe(elapsed.Seconds())
			}
		} else {
			riskScore = ScoreEvent(event.Amount)
		}

		metrics.RiskScores.Observe(float64(riskScore))

		processed := models.ProcessedEvent{
			EventID:     event.EventID,
			UserID:      event.UserID,
			EventType:   event.EventType,
			Amount:      event.Amount,
			RiskScore:   riskScore,
			Confidence:  confidence,
			MLScored:    mlScored,
			Explanation: explanation,
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
		logger.Info("processed event",
			"user_id", processed.UserID,
			"risk_score", processed.RiskScore,
			"confidence", processed.Confidence,
			"ml_scored", processed.MLScored,
		)
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

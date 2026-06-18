package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"

	internalKafka "github.com/apekshita/eventpulse/internal/kafka"
	"github.com/apekshita/eventpulse/internal/metrics"
	"github.com/apekshita/eventpulse/internal/models"
	"github.com/apekshita/eventpulse/internal/retry"
)

type GatewayHandler struct {
	Logger *slog.Logger
	DB     *sql.DB
	Writer internalKafka.MessageWriter
}

func (h *GatewayHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *GatewayHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, "/events", "405").Inc()
		return
	}

	var event models.Event
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&event); err != nil {
		h.Logger.Warn("invalid event payload", "error", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, "/events", "400").Inc()
		return
	}

	if event.EventID == "" {
		event.EventID = newEventID()
	}

	payload, err := json.Marshal(event)
	if err != nil {
		h.Logger.Error("failed to marshal event", "error", err)
		http.Error(w, "failed to encode event", http.StatusInternalServerError)
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, "/events", "500").Inc()
		return
	}

	if err := retry.Do(r.Context(), 5, 250*time.Millisecond, func() error {
		return h.Writer.WriteMessages(r.Context(), kafka.Message{Value: payload})
	}); err != nil {
		h.Logger.Error("kafka publish failed", "error", err)
		http.Error(w, "kafka unavailable", http.StatusServiceUnavailable)
		metrics.ProcessingErrors.WithLabelValues("api-gateway", "kafka_publish").Inc()
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, "/events", "503").Inc()
		return
	}

	metrics.EventsPublished.WithLabelValues("api-gateway", "events.raw").Inc()
	metrics.HTTPRequestsTotal.WithLabelValues(r.Method, "/events", "201").Inc()
	h.Logger.Info("published event", "user_id", event.UserID, "amount", event.Amount, "event_id", event.EventID)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Event Published"})
}

func (h *GatewayHandler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, "/alerts", "405").Inc()
		return
	}

	rows, err := h.DB.QueryContext(r.Context(), `SELECT id, user_id, risk_score, message, created_at FROM alerts ORDER BY id DESC`)
	if err != nil {
		h.Logger.Error("alert query failed", "error", err)
		http.Error(w, "failed to query alerts", http.StatusInternalServerError)
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, "/alerts", "500").Inc()
		return
	}
	defer rows.Close()

	alerts := make([]models.Alert, 0)
	for rows.Next() {
		var alert models.Alert
		if err := rows.Scan(&alert.ID, &alert.UserID, &alert.RiskScore, &alert.Message, &alert.CreatedAt); err != nil {
			h.Logger.Warn("alert row scan failed", "error", err)
			continue
		}
		alerts = append(alerts, alert)
	}

	if err := rows.Err(); err != nil {
		h.Logger.Error("alert row iteration failed", "error", err)
		http.Error(w, "failed to read alerts", http.StatusInternalServerError)
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, "/alerts", "500").Inc()
		return
	}

	metrics.HTTPRequestsTotal.WithLabelValues(r.Method, "/alerts", "200").Inc()
	writeJSON(w, http.StatusOK, alerts)
}

func (h *GatewayHandler) GetAlertByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, "/alert", "405").Inc()
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, "/alert", "400").Inc()
		return
	}

	var alert models.Alert
	err = h.DB.QueryRowContext(r.Context(),
		`SELECT id, user_id, risk_score, message, created_at FROM alerts WHERE id = $1`, id).
		Scan(&alert.ID, &alert.UserID, &alert.RiskScore, &alert.Message, &alert.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "alert not found", http.StatusNotFound)
			metrics.HTTPRequestsTotal.WithLabelValues(r.Method, "/alert", "404").Inc()
			return
		}
		h.Logger.Error("alert lookup failed", "error", err)
		http.Error(w, "failed to query alert", http.StatusInternalServerError)
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, "/alert", "500").Inc()
		return
	}

	metrics.HTTPRequestsTotal.WithLabelValues(r.Method, "/alert", "200").Inc()
	writeJSON(w, http.StatusOK, alert)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func newEventID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

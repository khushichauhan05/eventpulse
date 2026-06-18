package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/segmentio/kafka-go"

	"github.com/apekshita/eventpulse/internal/handlers"
	"github.com/apekshita/eventpulse/internal/logging"
)

// mockWriter satisfies kafka.MessageWriter without a real Kafka broker.
type mockWriter struct{ err error }

func (m *mockWriter) WriteMessages(_ context.Context, _ ...kafka.Message) error { return m.err }
func (m *mockWriter) Close() error                                               { return nil }

func newHandler(w *mockWriter) *handlers.GatewayHandler {
	return &handlers.GatewayHandler{
		Logger: logging.New("test", "INFO"),
		Writer: w,
	}
}

func TestHealth(t *testing.T) {
	h := newHandler(&mockWriter{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	h.Health(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %q, want %q", body["status"], "ok")
	}
}

func TestCreateEvent_MethodNotAllowed(t *testing.T) {
	h := newHandler(&mockWriter{})
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	rr := httptest.NewRecorder()

	h.CreateEvent(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
}

func TestCreateEvent_InvalidJSON(t *testing.T) {
	h := newHandler(&mockWriter{})
	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateEvent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestCreateEvent_Success(t *testing.T) {
	h := newHandler(&mockWriter{})
	body, _ := json.Marshal(map[string]any{
		"user_id":    "u1",
		"event_type": "purchase",
		"amount":     50000,
	})
	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateEvent(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rr.Code)
	}
}

func TestCreateEvent_EventIDAssignedWhenMissing(t *testing.T) {
	// If the client doesn't supply event_id, the gateway must generate one.
	published := make([]kafka.Message, 0)
	w := &captureWriter{msgs: &published}
	h := &handlers.GatewayHandler{
		Logger: logging.New("test", "INFO"),
		Writer: w,
	}

	body, _ := json.Marshal(map[string]any{"user_id": "u1", "event_type": "buy", "amount": 100})
	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.CreateEvent(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rr.Code)
	}

	if len(published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(published))
	}

	var event map[string]any
	if err := json.Unmarshal(published[0].Value, &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if event["event_id"] == "" || event["event_id"] == nil {
		t.Error("event_id should be set by gateway when not provided")
	}
}

type captureWriter struct{ msgs *[]kafka.Message }

func (c *captureWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	*c.msgs = append(*c.msgs, msgs...)
	return nil
}
func (c *captureWriter) Close() error { return nil }

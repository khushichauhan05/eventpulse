package services_test

import (
	"testing"

	"github.com/apekshita/eventpulse/internal/services"
)

func TestScoreEvent_LowRisk(t *testing.T) {
	cases := []float64{0, 1, 5000, 9999, 10000}
	for _, amount := range cases {
		score := services.ScoreEvent(amount)
		if score != 20 {
			t.Errorf("ScoreEvent(%.0f) = %d, want 20", amount, score)
		}
	}
}

func TestScoreEvent_HighRisk(t *testing.T) {
	cases := []float64{10001, 50000, 100000, 999999}
	for _, amount := range cases {
		score := services.ScoreEvent(amount)
		if score != 90 {
			t.Errorf("ScoreEvent(%.0f) = %d, want 90", amount, score)
		}
	}
}

func TestScoreEvent_BoundaryExact(t *testing.T) {
	if got := services.ScoreEvent(10000); got != 20 {
		t.Errorf("ScoreEvent(10000) = %d, want 20 (boundary is exclusive)", got)
	}
	if got := services.ScoreEvent(10001); got != 90 {
		t.Errorf("ScoreEvent(10001) = %d, want 90 (one above boundary)", got)
	}
}

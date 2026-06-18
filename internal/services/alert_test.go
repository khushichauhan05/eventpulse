package services_test

import (
	"testing"

	"github.com/apekshita/eventpulse/internal/services"
)

func TestIsHighRisk_Below(t *testing.T) {
	cases := []int{0, 20, 50, 79}
	for _, score := range cases {
		if services.IsHighRisk(score) {
			t.Errorf("IsHighRisk(%d) = true, want false", score)
		}
	}
}

func TestIsHighRisk_AtThreshold(t *testing.T) {
	if !services.IsHighRisk(80) {
		t.Error("IsHighRisk(80) = false, want true (boundary is inclusive)")
	}
}

func TestIsHighRisk_Above(t *testing.T) {
	cases := []int{81, 90, 95, 100}
	for _, score := range cases {
		if !services.IsHighRisk(score) {
			t.Errorf("IsHighRisk(%d) = false, want true", score)
		}
	}
}

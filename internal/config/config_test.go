package config_test

import (
	"testing"

	"github.com/apekshita/eventpulse/internal/config"
)

func TestLoad_DefaultsForAPIGateway(t *testing.T) {
	cfg := config.Load("api-gateway")

	if cfg.ServiceName != "api-gateway" {
		t.Errorf("ServiceName = %q, want %q", cfg.ServiceName, "api-gateway")
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.RawTopic != "events.raw" {
		t.Errorf("RawTopic = %q, want %q", cfg.RawTopic, "events.raw")
	}
	if cfg.ProcessedTopic != "events.processed" {
		t.Errorf("ProcessedTopic = %q, want %q", cfg.ProcessedTopic, "events.processed")
	}
	if cfg.AlertsTopic != "alerts" {
		t.Errorf("AlertsTopic = %q, want %q", cfg.AlertsTopic, "alerts")
	}
	if cfg.DLQTopic != "events.dlq" {
		t.Errorf("DLQTopic = %q, want %q", cfg.DLQTopic, "events.dlq")
	}
	if len(cfg.KafkaBrokers) == 0 {
		t.Error("KafkaBrokers should not be empty")
	}
}

func TestLoad_DefaultHealthPortsPerService(t *testing.T) {
	cases := []struct {
		service string
		want    string
	}{
		{"analytics-service", "8081"},
		{"alert-service", "8082"},
	}

	for _, tc := range cases {
		cfg := config.Load(tc.service)
		if cfg.HealthPort != tc.want {
			t.Errorf("service %q: HealthPort = %q, want %q", tc.service, cfg.HealthPort, tc.want)
		}
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("PORT", "9999")
	t.Setenv("KAFKA_TOPIC_DLQ", "my.dlq")

	cfg := config.Load("api-gateway")

	if cfg.Port != "9999" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9999")
	}
	if cfg.DLQTopic != "my.dlq" {
		t.Errorf("DLQTopic = %q, want %q", cfg.DLQTopic, "my.dlq")
	}
}

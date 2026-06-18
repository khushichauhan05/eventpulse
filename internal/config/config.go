package config

import (
	"os"
	"strings"
)

type Config struct {
	ServiceName    string
	Port           string
	HealthPort     string
	DatabaseDSN    string
	KafkaBrokers   []string
	RawTopic       string
	ProcessedTopic string
	AlertsTopic    string
	AnalyticsGroup string
	AlertGroup     string
	LogLevel       string
}

func Load(serviceName string) Config {
	return Config{
		ServiceName:    getEnv("SERVICE_NAME", serviceName),
		Port:           getEnv("PORT", defaultPort(serviceName)),
		HealthPort:     getEnv("HEALTH_PORT", defaultHealthPort(serviceName)),
		DatabaseDSN:    getEnv("DATABASE_DSN", defaultDatabaseDSN()),
		KafkaBrokers:   splitAndTrim(getEnv("KAFKA_BROKERS", "kafka:9092")),
		RawTopic:       getEnv("KAFKA_TOPIC_RAW", "events.raw"),
		ProcessedTopic: getEnv("KAFKA_TOPIC_PROCESSED", "events.processed"),
		AlertsTopic:    getEnv("KAFKA_TOPIC_ALERTS", "alerts"),
		AnalyticsGroup: getEnv("KAFKA_ANALYTICS_GROUP", "analytics-group"),
		AlertGroup:     getEnv("KAFKA_ALERT_GROUP", "alert-group"),
		LogLevel:       getEnv("LOG_LEVEL", "INFO"),
	}
}

func getEnv(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}

func splitAndTrim(value string) []string {
	parts := strings.Split(value, ",")
	brokers := make([]string, 0, len(parts))
	for _, part := range parts {
		broker := strings.TrimSpace(part)
		if broker != "" {
			brokers = append(brokers, broker)
		}
	}

	if len(brokers) == 0 {
		return []string{"kafka:9092"}
	}

	return brokers
}

func defaultPort(serviceName string) string {
	switch serviceName {
	case "api-gateway":
		return "8080"
	default:
		return "8080"
	}
}

func defaultHealthPort(serviceName string) string {
	switch serviceName {
	case "analytics-service":
		return "8081"
	case "alert-service":
		return "8082"
	default:
		return "8080"
	}
}

func defaultDatabaseDSN() string {
	return "host=postgres port=5432 user=admin password=admin123 dbname=eventpulse sslmode=disable"
}

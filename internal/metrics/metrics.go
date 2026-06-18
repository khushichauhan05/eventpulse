package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "eventpulse_http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	EventsPublished = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "eventpulse_events_published_total",
		Help: "Total events published to Kafka.",
	}, []string{"service", "topic"})

	EventsConsumed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "eventpulse_events_consumed_total",
		Help: "Total events consumed from Kafka.",
	}, []string{"service", "topic"})

	EventsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "eventpulse_events_processed_total",
		Help: "Total events successfully processed.",
	}, []string{"service"})

	AlertsGenerated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "eventpulse_alerts_generated_total",
		Help: "Total fraud alerts generated and stored.",
	})

	DLQMessages = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "eventpulse_dlq_messages_total",
		Help: "Total messages routed to the dead-letter queue.",
	}, []string{"service", "reason"})

	ProcessingErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "eventpulse_processing_errors_total",
		Help: "Total processing errors by stage.",
	}, []string{"service", "stage"})

	RiskScores = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "eventpulse_risk_score_distribution",
		Help:    "Distribution of risk scores assigned to events.",
		Buckets: []float64{10, 20, 30, 50, 70, 80, 90, 100},
	})
)

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

type ProcessedEvent struct {
	UserID    string  `json:"user_id"`
	EventType string  `json:"event_type"`
	Amount    float64 `json:"amount"`
	RiskScore int     `json:"risk_score"`
}

type Alert struct {
	UserID    string `json:"user_id"`
	Message   string `json:"message"`
	RiskScore int    `json:"risk_score"`
}

func main() {
	db, err := sql.Open(
		"postgres",
		"host=localhost port=5432 user=admin password=admin123 dbname=eventpulse sslmode=disable",
	)

	if err != nil {
		panic(err)
	}

	defer db.Close()

	fmt.Println("Connected to PostgreSQL")

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "events.processed",
		GroupID: "alert-group",
	})

	writer := &kafka.Writer{
		Addr:     kafka.TCP("localhost:9092"),
		Topic:    "alerts",
		Balancer: &kafka.LeastBytes{},
	}

	fmt.Println("Alert Service Started...")

	for {

		message, err := reader.ReadMessage(context.Background())
		if err != nil {
			fmt.Println("Read Error:", err)
			continue
		}

		var event ProcessedEvent

		err = json.Unmarshal(message.Value, &event)
		if err != nil {
			fmt.Println("JSON Error:", err)
			continue
		}

		if event.RiskScore >= 80 {

			alert := Alert{
				UserID:    event.UserID,
				Message:   "HIGH RISK TRANSACTION DETECTED",
				RiskScore: event.RiskScore,
			}

			alertBytes, _ := json.Marshal(alert)

			err = writer.WriteMessages(
				context.Background(),
				kafka.Message{
					Value: alertBytes,
				},
			)

			if err != nil {
				fmt.Println("Alert Publish Error:", err)
				continue
			}

			fmt.Println("ALERT GENERATED:")
			fmt.Println(string(alertBytes))
			_, err = db.Exec(
				`INSERT INTO alerts(user_id, risk_score, message)
	 VALUES($1,$2,$3)`,
				alert.UserID,
				alert.RiskScore,
				alert.Message,
			)

			if err != nil {
				fmt.Println("DB Insert Error:", err)
				continue
			}

			fmt.Println("Alert Stored In Database")
		}
	}
}

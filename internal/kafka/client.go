package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/apekshita/eventpulse/internal/retry"
)

func NewReader(brokers []string, topic, groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:         brokers,
		Topic:           topic,
		GroupID:         groupID,
		StartOffset:     kafka.FirstOffset,
		MinBytes:        1,
		MaxBytes:        10e6,
		MaxWait:         500 * time.Millisecond,
		CommitInterval:  0,
		ReadLagInterval: -1,
	})
}

func NewWriter(brokers []string, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		BatchTimeout: 10 * time.Millisecond,
		Async:        false,
	}
}

func WriteWithRetry(ctx context.Context, writer *kafka.Writer, value []byte) error {
	return retry.Do(ctx, 5, 250*time.Millisecond, func() error {
		return writer.WriteMessages(ctx, kafka.Message{Value: value})
	})
}

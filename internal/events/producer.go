package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Producer publishes reconciliation events to a Kafka topic.
// If Kafka is unavailable, all operations are no-ops.
type Producer struct {
	writer    *kafka.Writer
	available bool
}

// NewProducer creates a Kafka producer. If brokers is empty, the producer
// operates in no-op mode.
func NewProducer(brokers []string) *Producer {
	if len(brokers) == 0 || brokers[0] == "" {
		return &Producer{available: false}
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        Topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		Async:        true,
		RequiredAcks: kafka.RequireOne,
	}

	return &Producer{writer: w, available: true}
}

// Publish sends an event to Kafka. No-op if Kafka is unavailable.
func (p *Producer) Publish(ctx context.Context, event InstanceEvent) error {
	if !p.available {
		return nil
	}

	log := logf.FromContext(ctx).WithName("kafka-producer")

	event.Timestamp = time.Now()
	data, err := json.Marshal(event)
	if err != nil {
		log.Error(err, "failed to marshal event")
		return err
	}

	msg := kafka.Message{
		Key:   []byte(event.InstanceID),
		Value: data,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		log.Error(err, "failed to publish event", "type", event.Type, "instanceID", event.InstanceID)
		return err
	}

	log.V(1).Info("Published event", "type", event.Type, "instanceID", event.InstanceID)
	return nil
}

// Close shuts down the Kafka writer.
func (p *Producer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

// IsAvailable returns whether Kafka is connected.
func (p *Producer) IsAvailable() bool {
	return p.available
}

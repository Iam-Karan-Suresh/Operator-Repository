package events

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Consumer reads reconciliation events from Kafka and pushes them
// to a Go channel for the dashboard SSE handler to consume.
type Consumer struct {
	reader    *kafka.Reader
	available bool
	Events    chan InstanceEvent
}

// NewConsumer creates a Kafka consumer. If brokers is empty, the consumer
// operates in no-op mode.
func NewConsumer(brokers []string, groupID string) *Consumer {
	if len(brokers) == 0 || brokers[0] == "" {
		return &Consumer{
			available: false,
			Events:    make(chan InstanceEvent, 100),
		}
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    Topic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e6,
	})

	return &Consumer{
		reader:    r,
		available: true,
		Events:    make(chan InstanceEvent, 100),
	}
}

// Start begins consuming messages from Kafka and pushing them to the Events channel.
func (c *Consumer) Start(ctx context.Context) error {
	if !c.available {
		<-ctx.Done()
		return nil
	}

	log := logf.FromContext(ctx).WithName("kafka-consumer")
	log.Info("Starting Kafka consumer", "topic", Topic)

	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Info("Kafka consumer shutting down")
				return nil
			}
			log.Error(err, "failed to read message from Kafka")
			continue
		}

		var event InstanceEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Error(err, "failed to unmarshal event")
			continue
		}

		select {
		case c.Events <- event:
		default:
		}
	}
}

// Close shuts down the Kafka reader.
func (c *Consumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}

// IsAvailable returns whether Kafka is connected.
func (c *Consumer) IsAvailable() bool {
	return c.available
}

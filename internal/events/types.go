package events

import "time"

// EventType classifies the reconciliation event.
type EventType string

const (
	InstanceCreated EventType = "InstanceCreated"
	InstanceUpdated EventType = "InstanceUpdated"
	InstanceDeleted EventType = "InstanceDeleted"
	DriftDetected   EventType = "DriftDetected"
)

// Topic is the Kafka topic for operator events.
const Topic = "ec2-operator-events"

// InstanceEvent represents a reconciliation event published to Kafka.
type InstanceEvent struct {
	Type       EventType `json:"type"`
	InstanceID string    `json:"instanceID"`
	Name       string    `json:"name"`
	Namespace  string    `json:"namespace"`
	State      string    `json:"state"`
	PublicIP   string    `json:"publicIP,omitempty"`
	PrivateIP  string    `json:"privateIP,omitempty"`
	Region     string    `json:"region,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	Message    string    `json:"message,omitempty"`
}

package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestInstanceEventSerialization(t *testing.T) {
	event := InstanceEvent{
		Type:       InstanceCreated,
		InstanceID: "i-12345",
		Name:       "test-instance",
		Namespace:  "default",
		State:      "running",
		PublicIP:   "1.2.3.4",
		PrivateIP:  "10.0.0.1",
		Region:     "us-east-1",
		Timestamp:  time.Now(),
		Message:    "Instance created successfully",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	var decoded InstanceEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if decoded.Type != InstanceCreated {
		t.Errorf("expected type %s, got %s", InstanceCreated, decoded.Type)
	}
	if decoded.InstanceID != "i-12345" {
		t.Errorf("expected instanceID i-12345, got %s", decoded.InstanceID)
	}
}

func TestEventTypes(t *testing.T) {
	types := []EventType{InstanceCreated, InstanceUpdated, InstanceDeleted, DriftDetected}
	expected := []string{"InstanceCreated", "InstanceUpdated", "InstanceDeleted", "DriftDetected"}

	for i, typ := range types {
		if string(typ) != expected[i] {
			t.Errorf("expected %s, got %s", expected[i], typ)
		}
	}
}

func TestNoOpProducer(t *testing.T) {
	p := NewProducer(nil)

	if p.IsAvailable() {
		t.Fatal("expected no-op producer to be unavailable")
	}

	if err := p.Close(); err != nil {
		t.Fatalf("expected no error closing no-op producer: %v", err)
	}
}

func TestNoOpConsumer(t *testing.T) {
	c := NewConsumer(nil, "test-group")

	if c.IsAvailable() {
		t.Fatal("expected no-op consumer to be unavailable")
	}

	if err := c.Close(); err != nil {
		t.Fatalf("expected no error closing no-op consumer: %v", err)
	}
}

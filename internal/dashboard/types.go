package dashboard

import "time"

// InstanceResponse represents the flattened EC2 instance data sent to the frontend
type InstanceResponse struct {
	Name             string            `json:"name"`
	Namespace        string            `json:"namespace"`
	InstanceID       string            `json:"instanceID"`
	State            string            `json:"state"`
	PublicIP         string            `json:"publicIP"`
	PrivateIP        string            `json:"privateIP"`
	PublicDNS        string            `json:"publicDNS"`
	PrivateDNS       string            `json:"privateDNS"`
	InstanceType     string            `json:"instanceType"`
	AMIId            string            `json:"amiId"`
	Region           string            `json:"region"`
	AvailabilityZone string            `json:"availabilityZone,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
	CreatedAt        time.Time         `json:"createdAt"`
	Age              string            `json:"age"`
	Storage          StorageResponse   `json:"storage,omitempty"`
}

// StorageResponse represents the storage configuration for the UI
type StorageResponse struct {
	TotalSize         int32            `json:"totalSize"`
	RootVolume        VolumeResponse   `json:"rootVolume"`
	AdditionalVolumes []VolumeResponse `json:"additionalVolumes,omitempty"`
}

// VolumeResponse represents a single volume for the UI
type VolumeResponse struct {
	Size       int32  `json:"size"`
	Type       string `json:"type"`
	DeviceName string `json:"deviceName"`
}

// WatchEvent represents an SSE event sent to the frontend
type WatchEvent struct {
	Type   string           `json:"type"` // "ADDED", "MODIFIED", "DELETED"
	Object InstanceResponse `json:"object"`
}

// EventResponse represents a Kubernetes event
type EventResponse struct {
	Type    string    `json:"type"` // Normal, Warning
	Reason  string    `json:"reason"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
	Age     string    `json:"age"`
	Object  string    `json:"object"`
}

// LogResponse represents a line of log from the operator
type LogResponse struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Raw       string `json:"raw"`
}

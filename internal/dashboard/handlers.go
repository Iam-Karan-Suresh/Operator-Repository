package dashboard

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	computev1 "github.com/Iam-Karan-Suresh/operator-repo/api/v1"
	"github.com/Iam-Karan-Suresh/operator-repo/internal/controller"
	dto "github.com/prometheus/client_model/go"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	DefaultNamespace = "default"
)

type UISettings struct {
	Name       string `json:"name"`
	Profession string `json:"profession"`
	Team       string `json:"team"`
}

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods/log,verbs=get;list;watch

// Server handles dashboard API requests
type Server struct {
	client    client.Client
	clientset *kubernetes.Clientset
	port      string
	namespace string
	staticFS  fs.FS
}

func NewServer(mgrClient client.Client, clientset *kubernetes.Clientset, port string) *Server {
	// Try to get namespace from environment, fallback to default
	ns := os.Getenv("POD_NAMESPACE")
	if ns == "" {
		ns = DefaultNamespace
	}
	return &Server{
		client:    mgrClient,
		clientset: clientset,
		port:      port,
		namespace: ns,
	}
}

// SetNamespace sets the namespace for the dashboard settings
func (s *Server) SetNamespace(ns string) {
	s.namespace = ns
}

// SetStaticFS sets the static file system for the dashboard
func (s *Server) SetStaticFS(f fs.FS) {
	s.staticFS = f
}

// Start runs the HTTP server. It implements manager.Runnable
func (s *Server) Start(ctx context.Context) error {
	return s.StartWithFS(ctx, s.staticFS)
}

// StartWithFS runs the HTTP server and serves static files from the provided filesystem
func (s *Server) StartWithFS(ctx context.Context, f fs.FS) error {
	if f != nil {
		s.staticFS = f
	}
	l := log.FromContext(ctx)

	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("/api/instances", s.handleListInstances)
	mux.HandleFunc("/api/instances/", s.handleGetInstanceOrWatch) // prefixes with GET /api/instances/{name} or /api/instances/watch
	mux.HandleFunc("GET /api/instances/{namespace}/{name}/logs", s.handleInstanceLogs)
	mux.HandleFunc("GET /api/settings", s.handleGetSettings)
	mux.HandleFunc("POST /api/settings", s.handleUpdateSettings)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/healthz", s.handleHealthz)

	// Static File handling
	if s.staticFS != nil {
		mux.Handle("/", http.FileServer(http.FS(s.staticFS)))
	} else {
		mux.HandleFunc("/", s.handleStatic) // fallback for React SPA local testing
	}

	// Add CORS middleware
	handler := s.corsMiddleware(mux)

	srv := &http.Server{
		Addr:    s.port,
		Handler: handler,
	}

	// Start server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		l.Info("Starting dashboard server", "port", s.port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for context cancellation
	select {
	case <-ctx.Done():
		l.Info("Shutting down dashboard server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleListInstances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	var instances computev1.Ec2InstanceList

	if err := s.client.List(ctx, &instances); err != nil {
		log.FromContext(ctx).Error(err, "Failed to list instances")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := make([]InstanceResponse, 0, len(instances.Items))
	for _, inst := range instances.Items {
		response = append(response, mapToInstanceResponse(&inst))
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.FromContext(ctx).Error(err, "Failed to encode response")
	}
}

func (s *Server) handleGetInstanceOrWatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pathName := strings.TrimPrefix(r.URL.Path, "/api/instances/")

	if pathName == "watch" {
		s.handleWatchInstances(w, r)
		return
	}

	if instanceName, ok := strings.CutSuffix(pathName, "/events"); ok {
		s.handleGetEvents(w, r, instanceName)
		return
	}

	// It's a GET request for a specific instance
	if pathName == "" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// For simplicity, we assume default namespace in the dashboard, or we could pass ?namespace=foo
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default" // fallback
	}

	ctx := r.Context()
	var instance computev1.Ec2Instance
	if err := s.client.Get(ctx, client.ObjectKey{Name: pathName, Namespace: namespace}, &instance); err != nil {
		if errors.IsNotFound(err) {
			http.Error(w, "Instance not found", http.StatusNotFound)
			return
		}
		log.FromContext(ctx).Error(err, "Failed to get instance")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(mapToInstanceResponse(&instance)); err != nil {
		log.FromContext(ctx).Error(err, "Failed to encode response")
	}
}

func (s *Server) handleWatchInstances(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := log.FromContext(ctx).WithName("sse")

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// We use client.Watch if it's available, but controller-runtime client.Client
	// doesn't expose a direct Watch interface for arbitrary unstructured/typed watch easily
	// outside of cache/source.
	// For simplicity in the dashboard, we will poll the server for this example every X seconds,
	// or we can tap into the Kubernetes standard client. Let's do polling with a ticker for simplicity
	// and to avoid tying up K8s API watches directly per client if we don't have a cache configured.
	// Production dashboards typically use the controller-runtime Cache directly, or Informers.

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Keep track of previous state to only send updates when something changes
	previousState := make(map[string]InstanceResponse)

	// Context cancellation from client disconnect
	notify := r.Context().Done()

	for {
		select {
		case <-notify:
			l.Info("Client disconnected from SSE stream")
			return
		case <-ticker.C:
			var instances computev1.Ec2InstanceList
			if err := s.client.List(ctx, &instances); err != nil {
				l.Error(err, "Failed to list instances in watch loop")
				continue
			}

			currentMap := make(map[string]InstanceResponse)
			for _, inst := range instances.Items {
				resp := mapToInstanceResponse(&inst)
				key := resp.Namespace + "/" + resp.Name
				currentMap[key] = resp

				prev, exists := previousState[key]
				if !exists {
					// ADDED
					sendSSEEvent(w, flusher, "ADDED", resp)
				} else if prev.State != resp.State || prev.Age != resp.Age {
					// Age will update often, so maybe don't trigger on age alone unless you want 1s updates.
					// Let's only trigger on State or other meaningful fields changing
					if fmt.Sprintf("%v", prev) != fmt.Sprintf("%v", resp) {
						// MODIFIED
						sendSSEEvent(w, flusher, "MODIFIED", resp)
					}
				}
			}

			// Check for DELETED
			for key, prev := range previousState {
				if _, exists := currentMap[key]; !exists {
					sendSSEEvent(w, flusher, "DELETED", prev)
				}
			}
			previousState = currentMap
		}
	}
}

func sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data InstanceResponse) {
	event := WatchEvent{
		Type:   eventType,
		Object: data,
	}
	bytes, err := json.Marshal(event)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "data: %s\n\n", string(bytes))
	flusher.Flush()
}

func mapToInstanceResponse(inst *computev1.Ec2Instance) InstanceResponse {
	age := duration.HumanDuration(time.Since(inst.CreationTimestamp.Time))

	return InstanceResponse{
		Name:             inst.Name,
		Namespace:        inst.Namespace,
		InstanceID:       inst.Status.InstanceID,
		State:            inst.Status.State,
		PublicIP:         inst.Status.PublicIP,
		PrivateIP:        inst.Status.PrivateIP,
		PublicDNS:        inst.Status.PublicDNS,
		PrivateDNS:       inst.Status.PrivateDNS,
		InstanceType:     inst.Spec.InstanceType,
		AMIId:            inst.Spec.AMIId,
		Region:           inst.Spec.Region,
		AvailabilityZone: inst.Spec.AvailabilityZone,
		Tags:             inst.Spec.Tags,
		CreatedAt:        inst.CreationTimestamp.Time,
		Age:              age,
	}
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("DEBUG: Hit handleStatic for path: %s\n", r.URL.Path)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("React App Placeholder"))
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("DEBUG: Hit handleSettings for path: %s, method: %s\n", r.URL.Path, r.Method)
	switch r.Method {
	case http.MethodGet:
		s.handleGetSettings(w, r)
	case http.MethodPost:
		s.handleUpdateSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var cm corev1.ConfigMap
	err := s.client.Get(ctx, client.ObjectKey{Name: "ec2-operator-ui-settings", Namespace: s.namespace}, &cm)

	settings := UISettings{
		Name:       "User Name",
		Profession: "Project Lead",
		Team:       "Cloud Operations",
	}

	if err == nil {
		if val, ok := cm.Data["settings"]; ok {
			_ = json.Unmarshal([]byte(val), &settings)
		}
	}

	// Apply defaults if empty
	if settings.Name == "" {
		settings.Name = "User Name"
	}
	if settings.Profession == "" {
		settings.Profession = "Project Lead"
	}
	if settings.Team == "" {
		settings.Team = "Cloud Operations"
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(settings)
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var settings UISettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	data, err := json.Marshal(settings)
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to encode settings")
		http.Error(w, "Failed to encode settings", http.StatusInternalServerError)
		return
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ec2-operator-ui-settings",
			Namespace: s.namespace,
		},
		Data: map[string]string{
			"settings": string(data),
		},
	}

	var existing corev1.ConfigMap
	err = s.client.Get(ctx, client.ObjectKey{Name: "ec2-operator-ui-settings", Namespace: s.namespace}, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			log.FromContext(ctx).Info("Creating new settings ConfigMap", "namespace", s.namespace)
			if err := s.client.Create(ctx, cm); err != nil {
				if !errors.IsAlreadyExists(err) {
					log.FromContext(ctx).Error(err, "Failed to create settings ConfigMap", "namespace", s.namespace)
					http.Error(w, "Failed to create settings ConfigMap", http.StatusInternalServerError)
					return
				}
				// If it already exists, update below
			} else {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(settings)
				return
			}
		} else {
			log.FromContext(ctx).Error(err, "Failed to get existing settings ConfigMap", "namespace", s.namespace)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Update existing ConfigMap
	if existing.Name == "" {
		// Re-fetch in case of weirdness
		_ = s.client.Get(ctx, client.ObjectKey{Name: "ec2-operator-ui-settings", Namespace: s.namespace}, &existing)
	}

	existing.Data = cm.Data
	if err := s.client.Update(ctx, &existing); err != nil {
		log.FromContext(ctx).Error(err, "Failed to update settings ConfigMap", "namespace", s.namespace)
		http.Error(w, "Failed to update settings ConfigMap", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(settings)
}

type GlobalStats struct {
	ReconciliationCount int64   `json:"reconciliationCount"`
	InstanceCount       int     `json:"instanceCount"`
	ApiLatency          float64 `json:"apiLatency"`
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	// Get total reconciliations from Prometheus counter
	var m dto.Metric
	if err := controller.ReconciliationTotal.Write(&m); err != nil {
		http.Error(w, "Failed to read metrics", http.StatusInternalServerError)
		return
	}
	reconCount := int64(m.GetCounter().GetValue())

	// Get instance count from client
	var instances computev1.Ec2InstanceList
	if err := s.client.List(r.Context(), &instances); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get API Latency from Prometheus histogram
	var m2 dto.Metric
	latency := 0.0
	if err := controller.ApiLatency.Write(&m2); err == nil {
		count := m2.GetHistogram().GetSampleCount()
		if count > 0 {
			latency = m2.GetHistogram().GetSampleSum() / float64(count)
		}
	}

	stats := GlobalStats{
		ReconciliationCount: reconCount,
		InstanceCount:       len(instances.Items),
		ApiLatency:          latency * 1000, // convert to ms
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}
func (s *Server) handleGetEvents(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	// Fetch events for this specific object
	var eventList corev1.EventList

	// Create a selector for events involved with this specific Ec2Instance
	// The involvedObject.kind is Ec2Instance, and name is the instance name.
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingFields{
			"involvedObject.name": name,
			"involvedObject.kind": "Ec2Instance",
		},
	}

	if err := s.client.List(ctx, &eventList, listOpts...); err != nil {
		log.FromContext(ctx).Error(err, "Failed to list events")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var events []EventResponse
	for _, event := range eventList.Items {
		events = append(events, EventResponse{
			Type:    event.Type,
			Reason:  event.Reason,
			Message: event.Message,
			Time:    event.CreationTimestamp.Time,
			Age:     duration.HumanDuration(time.Since(event.CreationTimestamp.Time)),
			Object:  event.InvolvedObject.Kind + "/" + event.InvolvedObject.Name,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

func (s *Server) handleInstanceLogs(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/instances/"), "/")
	if len(parts) != 3 || parts[2] != "logs" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	namespace := parts[0]
	name := parts[1]

	// 1. Get operator pods
	pods, err := s.clientset.CoreV1().Pods(s.namespace).List(r.Context(), metav1.ListOptions{
		LabelSelector: "control-plane=controller-manager",
	})
	if err != nil || len(pods.Items) == 0 {
		http.Error(w, "operator pod not found", http.StatusInternalServerError)
		return
	}

	podName := pods.Items[0].Name

	// 2. Fetch logs
	tailLines := int64(1000)
	req := s.clientset.CoreV1().Pods(s.namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: "manager",
		TailLines: &tailLines,
	})
	podLogs, err := req.Stream(r.Context())
	if err != nil {
		log.FromContext(r.Context()).Error(err, "failed to stream logs")
		http.Error(w, "failed to stream logs", http.StatusInternalServerError)
		return
	}
	defer podLogs.Close()

	var logs []LogResponse
	scanner := bufio.NewScanner(podLogs)
	for scanner.Scan() {
		line := scanner.Text()
		// Filter by namespace and name
		if strings.Contains(line, name) && strings.Contains(line, namespace) {
			var logLine struct {
				Level string `json:"level"`
				TS    string `json:"ts"`
				Msg   string `json:"msg"`
			}
			err := json.Unmarshal([]byte(line), &logLine)
			if err == nil {
				logs = append(logs, LogResponse{
					Timestamp: logLine.TS,
					Level:     logLine.Level,
					Message:   logLine.Msg,
					Raw:       line,
				})
			} else {
				logs = append(logs, LogResponse{
					Timestamp: "",
					Level:     "info",
					Message:   line,
					Raw:       line,
				})
			}
		}
	}

	if logs == nil {
		logs = []LogResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

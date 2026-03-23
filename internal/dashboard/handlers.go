package dashboard

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strconv"
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
	DefaultNamespace = "operator-system"
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
	// Note: Standard net/http multiplexer doesn't support named parameters like {namespace} easily
	// without a router library or custom logic. For this embedded dashboard, we use simple prefix matching.
	mux.HandleFunc("/api/logs/", s.handleInstanceLogsLegacy) // Compatibility route
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
	for i := range instances.Items {
		response = append(response, mapToInstanceResponse(&instances.Items[i]))
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

	// Parsing /api/instances/{namespace}/{name}
	parts := strings.Split(pathName, "/")
	var ns, name string
	if len(parts) >= 2 {
		ns = parts[0]
		name = parts[1]
	} else {
		ns = r.URL.Query().Get("namespace")
		if ns == "" {
			ns = "default"
		}
		name = parts[0]
	}

	ctx := r.Context()
	var instance computev1.Ec2Instance
	if err := s.client.Get(ctx, client.ObjectKey{Name: name, Namespace: ns}, &instance); err != nil {
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

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	previousState := make(map[string]InstanceResponse)
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
			for i := range instances.Items {
				resp := mapToInstanceResponse(&instances.Items[i])
				key := resp.Namespace + "/" + resp.Name
				currentMap[key] = resp

				prev, exists := previousState[key]
				if !exists {
					sendSSEEvent(w, flusher, "ADDED", resp)
				} else if prev.State != resp.State || prev.Age != resp.Age {
					if fmt.Sprintf("%v", prev) != fmt.Sprintf("%v", resp) {
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
		Storage:          mapStorage(inst.Spec.Storage),
	}
}

func mapStorage(cfg computev1.StorageConfig) StorageResponse {
	root := VolumeResponse{
		Size:       cfg.RootVolume.Size,
		Type:       cfg.RootVolume.Type,
		DeviceName: cfg.RootVolume.DeviceName,
	}

	additional := make([]VolumeResponse, 0, len(cfg.AdditionalVolumes))
	total := root.Size
	for _, v := range cfg.AdditionalVolumes {
		total += v.Size
		additional = append(additional, VolumeResponse{
			Size:       v.Size,
			Type:       v.Type,
			DeviceName: v.DeviceName,
		})
	}

	return StorageResponse{
		TotalSize:         total,
		RootVolume:        root,
		AdditionalVolumes: additional,
	}
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("React App Placeholder"))
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
	TotalStorage        int64   `json:"totalStorage"`
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	var m dto.Metric
	if err := controller.ReconciliationTotal.Write(&m); err != nil {
		http.Error(w, "Failed to read metrics", http.StatusInternalServerError)
		return
	}
	reconCount := int64(m.GetCounter().GetValue())

	var instances computev1.Ec2InstanceList
	if err := s.client.List(r.Context(), &instances); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var m2 dto.Metric
	latency := 0.0
	if err := controller.ApiLatency.Write(&m2); err == nil {
		count := m2.GetHistogram().GetSampleCount()
		if count > 0 {
			latency = m2.GetHistogram().GetSampleSum() / float64(count)
		}
	}

	totalStorage := int64(0)
	for _, inst := range instances.Items {
		totalStorage += int64(inst.Spec.Storage.RootVolume.Size)
		for _, v := range inst.Spec.Storage.AdditionalVolumes {
			totalStorage += int64(v.Size)
		}
	}

	if reconCount == 0 || latency == 0 {
		remoteRecon, remoteLatency, err := s.fetchRemoteMetrics(r.Context())
		if err == nil {
			if remoteRecon > 0 {
				reconCount = remoteRecon
			}
			if remoteLatency > 0 {
				latency = remoteLatency
			}
		}
	}

	stats := GlobalStats{
		ReconciliationCount: reconCount,
		InstanceCount:       len(instances.Items),
		ApiLatency:          latency * 1000, // convert to ms
		TotalStorage:        totalStorage,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}
func (s *Server) fetchRemoteMetrics(ctx context.Context) (int64, float64, error) {
	if s.clientset == nil {
		return 0, 0, fmt.Errorf("clientset not available")
	}

	// Find operator pod
	log.FromContext(ctx).Info("Attempting to find operator pod for remote metrics", "namespace", s.namespace)
	pods, err := s.clientset.CoreV1().Pods(s.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "control-plane=controller-manager",
	})
	if err != nil || len(pods.Items) == 0 {
		// Try operator-system namespace if current one fails
		if s.namespace != "operator-system" {
			pods, err = s.clientset.CoreV1().Pods("operator-system").List(ctx, metav1.ListOptions{
				LabelSelector: "control-plane=controller-manager",
			})
		}
	}

	if err != nil || len(pods.Items) == 0 {
		return 0, 0, fmt.Errorf("operator pod not found")
	}

	podName := pods.Items[0].Name
	podNamespace := pods.Items[0].Namespace
	log.FromContext(ctx).Info("Found operator pod, proxying to metrics", "pod", podName, "namespace", podNamespace)

	// Proxy to metrics port (default 8080 or service port 8443)
	// Usually controller-runtime metrics are on 8080 inside the pod
	data, err := s.clientset.CoreV1().RESTClient().Get().
		Namespace(podNamespace).
		Resource("pods").
		SubResource("proxy").
		Name(fmt.Sprintf("%s:8080", podName)).
		Suffix("metrics").
		DoRaw(ctx)

	if err != nil {
		// Try 8443 if 8080 fails
		data, err = s.clientset.CoreV1().RESTClient().Get().
			Namespace(podNamespace).
			Resource("pods").
			SubResource("proxy").
			Name(fmt.Sprintf("%s:8443", podName)).
			Suffix("metrics").
			DoRaw(ctx)
	}

	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to proxy to remote metrics")
		return 0, 0, err
	}

	log.FromContext(ctx).Info("Successfully fetched remote metrics, parsing...", "bytes", len(data))

	var reconCount int64
	var latencySum float64
	var latencyCount int64

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ec2_operator_reconciliation_total") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				val, _ := strconv.ParseInt(parts[1], 10, 64)
				reconCount = val
			}
		} else if strings.HasPrefix(line, "ec2_operator_api_latency_seconds_sum") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				val, _ := strconv.ParseFloat(parts[1], 64)
				latencySum = val
			}
		} else if strings.HasPrefix(line, "ec2_operator_api_latency_seconds_count") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				val, _ := strconv.ParseInt(parts[1], 10, 64)
				latencyCount = val
			}
		}
	}

	avgLatency := 0.0
	if latencyCount > 0 {
		avgLatency = latencySum / float64(latencyCount)
	}

	return reconCount, avgLatency, nil
}

func (s *Server) handleGetEvents(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	var eventList corev1.EventList
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

	events := make([]EventResponse, 0, len(eventList.Items))
	for i := range eventList.Items {
		event := &eventList.Items[i]
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
	_ = json.NewEncoder(w).Encode(events)
}

func (s *Server) handleInstanceLogsLegacy(w http.ResponseWriter, r *http.Request) {
	// Fallback for older frontend versions
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	s.serveLogs(w, r, "default", parts[len(parts)-1])
}

func (s *Server) handleInstanceLogs(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/instances/"), "/")
	if len(parts) != 3 || parts[2] != "logs" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	s.serveLogs(w, r, parts[0], parts[1])
}

func (s *Server) serveLogs(w http.ResponseWriter, r *http.Request, namespace string, name string) {
	pods, err := s.clientset.CoreV1().Pods(s.namespace).List(r.Context(), metav1.ListOptions{
		LabelSelector: "control-plane=controller-manager",
	})
	if err != nil || len(pods.Items) == 0 {
		http.Error(w, "operator pod not found", http.StatusInternalServerError)
		return
	}

	podName := pods.Items[0].Name
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
	defer func() { _ = podLogs.Close() }()

	var logs []LogResponse
	scanner := bufio.NewScanner(podLogs)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, name) && strings.Contains(line, namespace) {
			var logLine struct {
				Level string `json:"level"`
				TS    string `json:"ts"`
				Msg   string `json:"msg"`
			}
			if err := json.Unmarshal([]byte(line), &logLine); err == nil {
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
	_ = json.NewEncoder(w).Encode(logs)
}

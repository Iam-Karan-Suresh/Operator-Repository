# Detailed Project Changelog & Architectural Evolution

## Core Transformations

### 1. From CRUD to Real-Time (Phase 5-6)
- **Initial**: Simple AWS SDK integration for create/delete.
- **Final**: Implemented a robust **SSE-based Real-time Dashboard**.
- **Architectural Shift**: Decoupled the UI from the polling-based reconciliation. The backend now acts as a high-frequency observer of the Kubernetes state, broadcasting events to the UI.

### 2. Security & AWS Identity (Phase 2-4)
- **Initial**: Hardcoded credentials in YAML.
- **Final**: Switched to **Environment-based AWS Identity**.
- **Architectural Shift**: Leverages standard `aws configure` or IAM Roles for Service Accounts (IRSA) in production, moving away from insecure secrets.

### 3. Unified Observability Stack (Phase 9-10)
- **Initial**: No monitoring.
- **Final**: Integrated **Prometheus, Grafana, Jaeger, and OpenCost**.
- **Architectural Shift**: Consolidated these disparate systems into a single Helm dependency graph. The operator is now a **fully instrumented observability hub**.

### 4. Personalization & Persistence (Phase 10-11)
- **Initial**: Static UI.
- **Final**: Implemented **ConfigMap-backed personalization**.
- **Architectural Shift**: Created a new API plane (`/api/settings`) where the UI can persist its own configuration directly into Kubernetes, treating the cluster as a state store for both infrastructure AND user preferences.

## All Changes (End-to-End)

| Phase | Milestone | Key Architectural Change |
|-------|-----------|-------------------------|
| 1 | Baseline Operator | Scaffolding with Kubebuilder |
| 2 | AWS SDK v2 | Integrated AWS SDK for Go v2 for EC2 lifecycle |
| 3 | State Reconciliation | Implemented drift detection logic |
| 4 | Security | Switched to Env-based AWS config |
| 5 | Dashboard Backend | Built embedded Go HTTP server with SSE |
| 6 | React UI | Built glassmorphism frontend with real-time patching |
| 7 | K8s Visibility | Added PrinterColumns for `kubectl get ec2instance` |
| 8 | Management UI | Added Bulk actions and advanced filters |
| 9 | Observability | Integrated OpenTelemetry, Jaeger, and OpenCost |
| 10 | Unified Helm | Consolidated stack into a single Chart |
| 11 | Metrics Polish | Added live stats API and dynamic metrics view |

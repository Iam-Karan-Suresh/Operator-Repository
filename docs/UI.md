# EC2 Operator Dashboard Documentation

## Overview
The Dashboard is a modern, real-time web interface built with React and Go, providing a visual way to manage and monitor EC2 instances across a Kubernetes cluster.

## Features
- **Real-time Updates**: Uses Server-Sent Events (SSE) via `/api/instances/watch` to stream status changes directly from the Kubernetes API to the browser.
- **Visual Lifecycle**: A custom `LifecycleTimeline` component tracks the instance through `pending`, `running`, `stopping`, `stopped`, and `terminated` states.
- **Cost Estimation**: Integrated **OpenCost** logic to estimate hourly/monthly costs based on instance types.
- **Bulk Operations**: Multi-select instances to perform start/stop/terminate actions across namespaces.
- **Personalization**: Customizable footer and sidebar tied to a persistent Kubernetes ConfigMap (`ec2-operator-ui-settings`).
- **Observability Integration**: Quick links and summary metrics for Prometheus, Grafana, and Jaeger traces.

## Technical Architecture
### 1. Backend (Go)
- **HTTP Server**: Implemented in `internal/dashboard/handlers.go`.
- **SSE Stream**: Active polling of the `client.Client` to detect resource events and broadcast them as JSON patches.
- **SPA Embedding**: The React production build is embedded into the Go binary using `embed.FS` for a zero-dependency deployment.

### 2. Frontend (React)
- **Vite & Tailwind**: Optimized build pipeline and a utility-first CSS approach for the dark-themed glassmorphism UI.
- **Lucide Icons**: Consistent iconography for system states.
- **Responsive Design**: Fully functional on mobile and desktop views.

## User Interface Sections
- **Instance List**: Search, filter by region/state, and view high-level tags and costs.
- **Instance Detail**: Deep dive into resource metadata (IPs, DNS, Placement) and visual state timeline.
- **Metrics View**: Real-time counters for reconciliations and active instances.
- **Settings View**: Global operator parameters and UI personalization (Your Name, Team).

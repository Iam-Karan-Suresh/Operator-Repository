# EC2 Instance Operator & Dashboard

A production-grade Open Source Kubernetes operator to manage AWS EC2 instances with a real-time, glassmorphism-styled dashboard and a full observability stack.

---

## ✨ Features

### 🛠 Core Operator Features
- **Declarative AWS EC2 Management**: Provision, update, and terminate instances using simple `Ec2Instance` Custom Resources.
- **Automatic Drift Detection**: Continuously monitors the state of your instances in AWS and reconciles any changes back to the desired Kubernetes state.
- **Unified Observability Stack**: Pre-configured integration with **Prometheus** (metrics), **Grafana** (dashboards), **Jaeger** (distributed tracing), and **OpenCost** (cost monitoring).
- **Production-Ready Helm Chart**: Easily deploy and manage the entire stack with a single Helm command.
- **Secure by Design**: Supports isolated AWS credential management via Kubernetes secrets.

### 🎨 Web UI Dashboard Features
- **Premium Glassmorphism Design**: A stunning, modern interface with smooth animations and interactive components.
- **Real-Time Streaming Updates**: Uses **Server-Sent Events (SSE)** to provide instantaneous updates on instance status without manual page refreshes.
- **Live Metrics Dashboard**: Visual representation of operator health, including total reconciliations and AWS API latency stats.
- **Integrated Instance Logs**: View real-time logs of the reconciliation process directly from the UI, similar to ArgoCD.
- **UI Personalization**: Save user preferences (Name, Profession, Team) that persist across sessions using Kubernetes ConfigMaps.
- **Responsive Layout**: Optimized for both ultra-wide monitors and standard displays.

---

## 🚀 Getting Started (Installation Guide)

### 1. Prerequisites
- **Minikube** (or any K8s cluster) installed and running.
- **Helm v3+** and **AWS CLI** configured (`aws configure`).

### 2. Configure AWS Access
The operator requires AWS credentials. You can provide them in two ways:

#### **Option A: Direct values in Helm (Fastest)**
During installation, set the values directly via the CLI:
```bash
--set awsCredentials.accessKeyId="YOUR_KEY" --set awsCredentials.secretAccessKey="YOUR_SECRET"
```

#### **Option B: Using a Kubernetes Secret (Recommended)**
Create a secret in the target namespace before installing:
```bash
kubectl create namespace operator-system
kubectl create secret generic aws-credentials \
  --from-literal=AWS_ACCESS_KEY_ID=YOUR_KEY \
  --from-literal=AWS_SECRET_ACCESS_KEY=YOUR_SECRET \
  -n operator-system
```

### 3. Install via Helm (OCI Registry)
Install the operator directly from the GitHub Container Registry:

```bash
# Create namespace if it doesn't exist
kubectl create namespace operator-system

# Install the operator and the full observability stack
helm install ec2-operator oci://ghcr.io/iam-karan-suresh/charts/ec2-operator \
  --version 1.1.1 \
  -n operator-system \
  --set awsCredentials.region="us-east-1"
```

### 4. Upgrade the Operator
To upgrade to the latest version:
```bash
helm upgrade ec2-operator oci://ghcr.io/iam-karan-suresh/charts/ec2-operator \
  --version 1.1.1 \
  -n operator-system
```

---

## 🏗 Developer & Maintenance Commands

### Build and Push Dashboard Image
If you are modifying the UI, rebuild and push the container image:
```bash
docker build -t docker.io/karanwebdev/ec2-dashboard:v1.1.1 -f Dockerfile.dashboard .
docker push docker.io/karanwebdev/ec2-dashboard:v1.1.1
```

### Package and Push Helm Chart
Package the chart and push it to the OCI registry:
```bash
helm package dist/chart
helm push ec2-operator-1.1.1.tgz oci://ghcr.io/iam-karan-suresh/charts
```

### Create GHCR Pull Secret
If you use a private repository for your images:
```bash
kubectl create secret docker-registry ghcr-pull-secret \
  --docker-server=ghcr.io \
  --docker-username="<GH_USERNAME>" \
  --docker-password="<GH_TOKEN>" \
  -n operator-system
```

---

## 📊 Accessing the Stack

### Dashboard UI
Expose the dashboard locally:
```bash
kubectl port-forward svc/operator-dashboard -n operator-system 3000:3000
```
Open **[http://localhost:3000](http://localhost:3000)**.

### Operator Logs
Monitor the operator's backend reconciliation activity:
```bash
kubectl logs -n operator-system -l control-plane=controller-manager -c manager --tail=100
```

### Observability Tools
- **Grafana**: `kubectl port-forward svc/operator-grafana -n operator-system 8080:80` (Go to http://localhost:8080)
- **Jaeger Tracing**: View AWS SDK traces at `http://localhost:16686` (via dashboard or port-forward).
- **Prometheus**: Raw metrics available via `operator-prometheus-server`.

---

## 🛠 Usage Example
Create your first instance by applying this YAML:

```yaml
apiVersion: compute.cloud.com/v1
kind: Ec2Instance
metadata:
  name: prod-web-server
spec:
  region: "us-east-1"
  amiID: "ami-0c55b159cbfafe1f0" # Amazon Linux 2
  instanceType: "t3.micro"
  volume: 25 # (Optional) Root volume size in GB
```

---

## 📖 Deep Dives
For more internal details, refer to:
- [Operator Architecture & Flow](docs/OPERATOR.md)
- [UI Features & Design](docs/UI.md)
- [System Architecture](docs/ARCHITECTURE.md)
- [Changelog & Evolution](docs/CHANGELOG.md)

---

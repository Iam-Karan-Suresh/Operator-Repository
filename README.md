# EC2 Instance Operator & Dashboard

A production-grade Kubernetes operator to manage AWS EC2 instances with a real-time, glassmorphism-styled dashboard and full observability stack.

## 🚀 Quick Start (Minikube / Local)

### 1. Prerequisites
- **Minikube** installed and running
- **Helm** v3+
- **AWS CLI** configured (`aws configure`)

### 2. Configure AWS Access
The operator uses your local AWS credentials. Ensure you have the necessary permissions for EC2 (e.g., `AmazonEC2FullAccess`).

### 3. Deploy the Stack
Deploy the operator, dashboard, and full observability stack with a single command:

```bash
# From the root of the repository
helm upgrade --install operator dist/chart/
```

### 4. Access the Dashboard
Expose the dashboard service to your local machine:

```bash
kubectl port-forward svc/operator-dashboard 3000:3000
```
Open [http://localhost:3000](http://localhost:3000) in your browser.

## 📊 Features
- **Declarative AWS EC2 Management**: Provision instances via `Ec2Instance` CRDs.
- **Real-time Visualization**: Glassmorphism UI with Server-Sent Events (SSE).
- **Drift Detection**: Automatically syncs Kubernetes state with AWS changes.
- **Full Observability**: Integrated Prometheus, Grafana, Jaeger (Tracing), and OpenCost.
- **Personalized Experience**: Save your name and team preferences directly to the cluster.

## 📖 Component Documentation
- [Operator Architecture & Flow](docs/OPERATOR.md)
- [UI Features & Design](docs/UI.md)
- [System Architecture](docs/ARCHITECTURE.md)
- [Changelog & Evolution](docs/CHANGELOG.md)

## 🛠 Usage Example
Create your first instance:
```yaml
apiVersion: compute.cloud.com/v1
kind: Ec2Instance
metadata:
  name: prod-web-server
spec:
  region: "us-east-1"
  amiID: "ami-0c55b159cbfafe1f0" # Amazon Linux 2
  instanceType: "t3.micro"
```

## 📜 Metrics & Tracing
- **Grafana**: Access dashboards via `kubectl port-forward svc/operator-grafana 8080:80`.
- **Jaeger**: View AWS SDK traces at `http://localhost:16686`.
- **Metrics**: Real-time stats available in the Metrics tab of the dashboard.

---
Built by **Antigravity Team**

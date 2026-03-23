# EC2 Instance Operator & Dashboard

A production-grade Kubernetes operator to manage AWS EC2 instances with a real-time, glassmorphism-styled dashboard and full observability stack.

## 🚀 Quick Start (Minikube / Local)

### 1. Prerequisites
- **Minikube** installed and running
- **Helm** v3+
- **AWS CLI** configured (`aws configure`)

### 2. Configure AWS Access
The operator requires AWS credentials to manage EC2 instances. You can provide them in two ways:

#### Option A: Direct values in `values.yaml` (Local Development)
Edit `dist/chart/values.yaml`:
```yaml
awsCredentials:
  accessKeyId: "YOUR_ACCESS_KEY"
  secretAccessKey: "YOUR_SECRET_KEY"
  region: "us-east-1"
```

#### Option B: Using a Kubernetes Secret (Recommended)
1. Create the secret:
```bash
kubectl create secret generic aws-credentials \
  --from-literal=AWS_ACCESS_KEY_ID=YOUR_ACCESS_KEY \
  --from-literal=AWS_SECRET_ACCESS_KEY=YOUR_SECRET_KEY
```
2. Update `dist/chart/values.yaml` to use the secret:
```yaml
awsCredentials:
  secretName: "aws-credentials"
  region: "us-east-1"
```

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

for logs of operator

```bash
kubectl logs -n operator-system operator-controller-manager-6b458f9767-t7xmv -c manager

```

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

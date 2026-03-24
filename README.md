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
  --from-literal=AWS_SECRET_ACCESS_KEY=YOUR_SECRET_KEY -n operator-system
```
2. Update `dist/chart/values.yaml` to use the secret:
```yaml
awsCredentials:
  secretName: "aws-credentials"
  region: "us-east-1"
```

### 3. Deploy via OCI Registry (Global)
Install the operator directly from the GitHub Container Registry:

```bash
# Add namespace
kubectl create namespace operator-system

# Install the operator and observability stack
helm install ec2-operator oci://ghcr.io/iam-karan-suresh/charts/ec2-operator \
  --version 1.1.1 \
  -n operator-system \
  --create-namespace \
  --set awsCredentials.accessKeyId="YOUR_ACCESS_KEY" \
  --set awsCredentials.secretAccessKey="YOUR_SECRET_KEY"
```

### 4. Upgrade an Existing Installation
```bash
helm upgrade ec2-operator oci://ghcr.io/iam-karan-suresh/charts/ec2-operator \
  --version 1.1.1 \
  -n operator-system
```

docker build -t docker.io/karanwebdev/ec2-dashboard:v1.1.1 -f Dockerfile.dashboard . && docker push docker.io/karanwebdev/ec2-dashboard:v1.1.1


helm package dist/chart && helm push ec2-operator-1.1.1.tgz oci://ghcr.io/iam-karan-suresh/ec2-operator


kubectl create secret docker-registry ghcr-pull-secret \
  --docker-server=ghcr.io \
  --docker-username="<YOUR_GITHUB_USERNAME>" \
  --docker-password="<YOUR_GITHUB_TOKEN>" \
  -n operator-system

### 4. Access the Dashboard
Expose the dashboard service to your local machine:

```bash
kubectl port-forward svc/operator-dashboard 3000:3000
```
Open [http://localhost:3000](http://localhost:3000) in your browser.

for logs of operator

```bash
 kubectl logs -n operator-system -l control-plane=controller-manager -c manager --tail=50

```

## 📊 Features
- **Declarative AWS EC2 Management**: Provision instances via `Ec2Instance` CRDs.
- **Real-time Visualization**: Glassmorphism UI with Server-Sent Events (SSE).
- **Drift Detection**: Automatically syncs Kubernetes state with AWS changes.
- **Full Observability**: Integrated Prometheus, Grafana, Jaeger (Tracing), and OpenCost.
- **Personalized Experience**: Save your name and team preferences directly to the cluster.

## 🔌 Frontend UI Architecture & Real-Time Updates

The dashboard provides a seamless, real-time view of EC2 instances without requiring manual refreshes.

### 🏗 Data Flow Architecture
1. **Kubernetes API Watch**: The backend dashboard server (written in Go) manages a persistent connection to the Kubernetes API.
2. **Server-Sent Events (SSE)**: Instead of traditional polling from the browser, we use a single long-lived HTTP connection (`/api/instances/watch`) to stream updates using Server-Sent Events.
3. **State Comparison Engine**:
   - The backend polls the Kubernetes API for `Ec2Instance` resources every **2 seconds**.
   - It maintains a `previousState` map for each connected client session.
   - It performs a deep comparison between the cluster state and the previous state to detect `ADDED`, `MODIFIED`, or `DELETED` events.
4. **React State Reconciliation**:
   - The `useInstances` hook in the React frontend maintains the global `instances` state.
   - When a `WatchEvent` is received over SSE, the hook performs an incremental update to the state array.
   - React automatically re-renders only the necessary UI components, ensuring smooth transitions and animations.

### 📡 Technical Stack
- **Streaming Protocol**: Server-Sent Events (SSE) for low-overhead, one-way streaming.
- **Backend**: Go `net/http` with `http.Flusher` for immediate data transmission.
- **Frontend**: React Hooks (`useEffect`, `useCallback`) and the native `EventSource` API.

### 🛠 Key Files for Developers
- **Backend Logic**: `internal/dashboard/handlers.go` (see `handleWatchInstances`)
- **API Client**: `web/src/api/client.ts`
- **Frontend Hook**: `web/src/hooks/useInstances.ts`

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

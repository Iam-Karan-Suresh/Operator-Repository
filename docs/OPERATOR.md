# EC2 Instance Operator Documentation

## Overview
The EC2 Instance Operator is a Kubernetes-native controller designed to manage lifecycle of AWS EC2 instances via Custom Resources (CRDs). It provides a declarative way to provision, update, and terminate EC2 instances directly from Kubernetes.

## Core Components
### 1. Controllers
- **Ec2InstanceReconciler**: The main reconciliation loop (`internal/controller/ec2instance_controller.go`).
  - Watches for changes to `Ec2Instance` resources in the `compute.cloud.com/v1` API.
  - Handles creation, deletion, and drift detection.
  - Implements finalizers for safe cleanup of AWS resources.

### 2. AWS SDK Integration
The operator uses the AWS SDK for Go v2 to interact with EC2 APIs.
- **Create**: Handled in `createInstance.go`. Uses `RunInstances` and waits for state transitions.
- **Delete**: Handled in `deleteInstance.go`. Uses `TerminateInstances`.
- **Check/Drift**: Handled in `checkInstance.go`. Uses `DescribeInstances` to verify the state in AWS vs. Kubernetes status.

### 3. Reconciliation Flow
1. **Trigger**: User applies an `Ec2Instance` YAML or an existing resource is updated/periodically polled.
2. **Retrieve**: Controller fetches the latest object from the Kubernetes API.
3. **Analyze**: 
   - If `DeletionTimestamp` is set, trigger AWS termination and remove finalizer.
   - If `InstanceID` is missing, trigger AWS instance creation.
   - If `InstanceID` exists, perform **Drift Detection**: Compare AWS state (running, stopped, etc.) with Kubernetes Status.
4. **Update**: Sync AWS metadata (IPs, DNS, State) back to the Kubernetes object's `.status` field.
5. **Wait**: Requeue for periodic health checks (default 30s).

## Metrics & Observability
- **Custom Metrics**: 
  - `ec2_operator_managed_instances_total`: Gauge tracking total running instances.
  - `ec2_operator_reconciliation_total`: Counter tracking total reconciliation attempts.
- **OpenTelemetry**: Integrated for distributed tracing. Every AWS interaction creates a span visible in Jaeger.

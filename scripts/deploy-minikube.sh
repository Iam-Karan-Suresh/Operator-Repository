#!/bin/bash

# EC2 Operator Minikube Deployment Script
# This script starts minikube with specialized resources, builds local code images,
# matches them with the local helm chart, and deploys everything to the cluster.

set -e

# Configuration
CLUSTER_NAME="ec2-operator-cluster"
CPUS=6
MEMORY=6144 # 6GB in MB
IMAGE_NAME="ec2-operator"
DASHBOARD_IMAGE_NAME="ec2-dashboard"
TAG="local"
CHART_PATH="./dist/chart"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== 🚀 EC2 Operator Minikube Detailed Deployment ===${NC}"

# 1. Dependency Checks
echo -e "${YELLOW}🔍 Checking dependencies...${NC}"
for cmd in minikube helm docker kubectl; do
    if ! command -v $cmd &> /dev/null; then
        echo -e "${RED}❌ Error: $cmd is not installed.${NC}"
        exit 1
    fi
done
echo -e "${GREEN}✅ Dependencies verified.${NC}"

# 2. Start Minikube Cluster
if minikube status -p "$CLUSTER_NAME" &> /dev/null; then
    echo -e "${GREEN}✅ Minikube cluster '$CLUSTER_NAME' already exists and is running.${NC}"
else
    echo -e "${BLUE}🚀 Starting Minikube cluster: $CLUSTER_NAME...${NC}"
    echo -e "   - CPUs: $CPUS"
    echo -e "   - RAM:  $MEMORY MB"
    minikube start \
        --cpus="$CPUS" \
        --memory="$MEMORY" \
        --driver=docker \
        -p "$CLUSTER_NAME"
fi

# 3. Environment Setup
echo -e "${BLUE}🐳 Pointing Docker context to Minikube daemon...${NC}"
# This ensures that 'docker build' pushes images directly into the cluster's registry
eval $(minikube -p "$CLUSTER_NAME" docker-env)

# 4. Local Builds
echo -e "${BLUE}🔨 Building Manager image from local source...${NC}"
# This builds the main Go operator + embedded React frontend
docker build -f Dockerfile -t "$IMAGE_NAME:$TAG" .

echo -e "${BLUE}🔨 Building Dashboard image from local source...${NC}"
# Optional separate dashboard binary if needed by the chart
docker build -f Dockerfile.dashboard -t "$DASHBOARD_IMAGE_NAME:$TAG" .

# 5. Helm Preparation
echo -e "${BLUE}📦 Updating Helm chart dependencies...${NC}"
helm dependency update "$CHART_PATH"

# 6. AWS Credentials Check (Requirement for Operator)
if ! kubectl get secret aws-credentials &> /dev/null; then
    echo -e "${YELLOW}⚠️  Warning: 'aws-credentials' secret not found.${NC}"
    echo -e "   The operator requires AWS credentials to function."
    echo -e "   Run the following if you have your keys ready:"
    echo -e "   kubectl create secret generic aws-credentials \\"
    echo -e "     --from-literal=AWS_ACCESS_KEY_ID=YOUR_KEY \\"
    echo -e "     --from-literal=AWS_SECRET_ACCESS_KEY=YOUR_SECRET"
fi

# 7. Deployment
echo -e "${BLUE}🚀 Deploying to Minikube using local Helm chart...${NC}"
# We override the repository and tag to use our local images
# We set imagePullPolicy=Never to prevent K8s from trying to pull from external registries
helm upgrade --install ec2-operator "$CHART_PATH" \
    --set controllerManager.container.image.repository="$IMAGE_NAME" \
    --set controllerManager.container.image.tag="$TAG" \
    --set controllerManager.container.imagePullPolicy=Never \
    --set dashboard.image.repository="$DASHBOARD_IMAGE_NAME" \
    --set dashboard.image.tag="$TAG" \
    --set dashboard.imagePullPolicy=Never \
    --wait

echo -e "${GREEN}✅ Everything is up and running in Minikube!${NC}"

# 8. Service Access
echo -e "\n${BLUE}📍 Useful URLs (Access relative to your host):${NC}"
dashboard_url=$(minikube -p "$CLUSTER_NAME" service operator-dashboard --url || echo "Retrieving...")
prometheus_url=$(minikube -p "$CLUSTER_NAME" service prometheus-server --url || echo "Retrieving...")
grafana_url=$(minikube -p "$CLUSTER_NAME" service grafana --url || echo "Retrieving...")
jaeger_url=$(minikube -p "$CLUSTER_NAME" service jaeger-query --url || echo "Retrieving...")

echo -e "   - Dashboard UI:  $dashboard_url"
echo -e "   - Prometheus:    $prometheus_url"
echo -e "   - Grafana:       $grafana_url (admin/admin)"
echo -e "   - Jaeger:        $jaeger_url"

echo -e "\n${YELLOW}💡 Note: To interact with the cluster via CLI, ensure you're in the right context:${NC}"
echo -e "   minikube -p \"$CLUSTER_NAME\" status"

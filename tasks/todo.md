# Dynamic Per-Instance Cost Visualization with OpenCost + AWS

## Objectives
- Dynamically fetch per-EC2 instance cost using OpenCost + AWS
- Eliminate static/hardcoded mappings
- Reduce latency significantly
- Upgrade UI to production-grade clarity
- Ensure accurate mapping between Kubernetes nodes ↔ EC2 instances

## Implementation Plan

### 1. Dynamic Instance Detection
- [x] Implement `extractInstanceDetails(providerID string)` to extract instanceId and region from Kubernetes node `spec.providerID`

### 2. OpenCost Data Mapping
- [x] Call OpenCost `/allocation?aggregate=node&window=1d` API
- [x] Map `node.Name` with OpenCost `node` field
- [x] Attach cost to `instanceId`

### 3. AWS Enrichment
- [x] Use AWS SDK (Go) `ec2.DescribeInstances()` to fetch `instanceType`, `region`, and `state`

### 4. Backend API
- [x] Design and implement `GET /api/cost/instances` returning JSON array of instance costs and metadata
- [x] Add endpoint `GET /api/cost/instances/{id}`

### 5. Latency Optimization
- [x] Implement in-memory cache (e.g. `sync.Map` or `go-cache`) with 30-60s TTL
- [x] Implement background refresh worker to fetch OpenCost data periodically
- [x] Process all instances in a single OpenCost call (Avoid N+1)
- [x] Use `sync.WaitGroup` for parallel AWS `DescribeInstances` calls

### 6. UI Enhancements
- [x] Update frontend to fetch `/api/cost/instances/{id}` when clicking an instance
- [x] Design a clean card layout showing: Instance ID, Cost (daily + monthly), Region, Instance Type
- [x] Add loading skeleton, fallback if cost missing, and tooltip "Cost calculated via OpenCost"
- [x] Implement color-coded cost levels and currency formatting

### 7. Reliability and Observability
- [x] Error handling: return cached data if OpenCost is down, or degraded mode if AWS fails
- [x] Add logs for OpenCost and AWS API latency
- [x] Add metrics for cache hit rate and API response time

## Review & Verification
- [x] Verify latency is <200ms
- [x] Check accurate costs show in UI
- [x] Test API responses and error modes

package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	computev1 "github.com/Iam-Karan-Suresh/operator-repo/api/v1"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// InstanceCostData represents the final enriched cost metadata
type InstanceCostData struct {
	InstanceID   string  `json:"instanceId"`
	Node         string  `json:"node"`
	InstanceType string  `json:"instanceType"`
	Region       string  `json:"region"`
	DailyCost    float64 `json:"dailyCost"`
	MonthlyCost  float64 `json:"monthlyCost"`
	State        string  `json:"state"`
}

// CostService handles syncing data from OpenCost and AWS
type CostService struct {
	clientSet   *kubernetes.Clientset
	k8sClient   client.Client
	opencostURL string
	cache       sync.Map
	syncPeriod  time.Duration
}

var (
	instanceHourlyCost = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ec2_operator_instance_hourly_cost_usd",
			Help: "Estimated hourly cost of an EC2 instance in USD",
		},
		[]string{"instance_id", "instance_type", "region", "namespace", "instance_name"},
	)
	instanceCumulativeCost = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ec2_operator_instance_cumulative_cost_usd",
			Help: "Estimated cumulative monthly cost of an EC2 instance in USD",
		},
		[]string{"instance_id", "instance_type", "region", "namespace", "instance_name"},
	)
)

func init() {
	metrics.Registry.MustRegister(instanceHourlyCost, instanceCumulativeCost)
}

// NewCostService creates a new background worker to fetch cost and AWS metadata
func NewCostService(k8sClient client.Client, clientSet *kubernetes.Clientset) *CostService {
	url := os.Getenv("OPENCOST_URL")
	if url == "" {
		url = "http://opencost.opencost.svc.cluster.local:9003"
	}
	return &CostService{
		k8sClient:   k8sClient,
		clientSet:   clientSet,
		opencostURL: url,
		syncPeriod:  60 * time.Second,
	}
}

// ec2OnDemandHourlyPriceUSD maps common instance types to their hourly on-demand
// Linux pricing in us-east-1 (as of 2024). Used as a fallback when OpenCost is unavailable.
var ec2OnDemandHourlyPriceUSD = map[string]float64{
	// t3 family
	"t3.nano":    0.0052,
	"t3.micro":   0.0104,
	"t3.small":   0.0208,
	"t3.medium":  0.0416,
	"t3.large":   0.0832,
	"t3.xlarge":  0.1664,
	"t3.2xlarge": 0.3328,
	// t2 family
	"t2.nano":    0.0058,
	"t2.micro":   0.0116,
	"t2.small":   0.023,
	"t2.medium":  0.0464,
	"t2.large":   0.0928,
	"t2.xlarge":  0.1856,
	"t2.2xlarge": 0.3712,
	// m5 family
	"m5.large":    0.096,
	"m5.xlarge":   0.192,
	"m5.2xlarge":  0.384,
	"m5.4xlarge":  0.768,
	"m5.8xlarge":  1.536,
	"m5.12xlarge": 2.304,
	// c5 family
	"c5.large":   0.085,
	"c5.xlarge":  0.17,
	"c5.2xlarge": 0.34,
	"c5.4xlarge": 0.68,
	// r5 family
	"r5.large":   0.126,
	"r5.xlarge":  0.252,
	"r5.2xlarge": 0.504,
	"r5.4xlarge": 1.008,
}

// dailyCostFromInstanceType returns a per-day fallback cost estimate based on instance type.
func dailyCostFromInstanceType(instanceType string) float64 {
	if price, ok := ec2OnDemandHourlyPriceUSD[instanceType]; ok {
		return price * 24
	}
	// Unknown type → 0 (will show as $0.00 rather than a spinning placeholder)
	return 0
}

// StartSync begins the background routine
func (s *CostService) StartSync(ctx context.Context) {
	l := log.FromContext(ctx).WithName("cost-service")
	l.Info("Starting background cost sync", "interval", s.syncPeriod)

	ticker := time.NewTicker(s.syncPeriod)
	defer ticker.Stop()

	// Initial sync
	s.syncData(ctx)

	for {
		select {
		case <-ctx.Done():
			l.Info("Stopping cost sync")
			return
		case <-ticker.C:
			s.syncData(ctx)
		}
	}
}

// extractInstanceDetails given aws:///us-east-1/i-12345 returns instanceId and region
func extractInstanceDetails(providerID string) (instanceID string, region string) {
	if !strings.HasPrefix(providerID, "aws://") {
		return "", ""
	}
	parts := strings.Split(providerID, "/")
	if len(parts) >= 2 {
		instanceID = parts[len(parts)-1]
		region = parts[len(parts)-2]
	}
	return
}

func (s *CostService) syncData(ctx context.Context) {
	l := log.FromContext(ctx).WithName("cost-service-sync")

	// 1. List K8s Nodes
	nodes, err := s.clientSet.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		l.Error(err, "Failed to list nodes")
		return
	}

	nodeMap := make(map[string]corev1.Node) // nodeName -> Node
	var instanceIDs []string
	instanceRegionMap := make(map[string]string) // instanceId -> region

	for _, n := range nodes.Items {
		nodeMap[n.Name] = n
		instanceID, region := extractInstanceDetails(n.Spec.ProviderID)
		if instanceID != "" {
			instanceIDs = append(instanceIDs, instanceID)
			instanceRegionMap[instanceID] = region
		}
	}

	// 1.5 List EC2Instances CRDs to find all instances we should track
	var instances computev1.Ec2InstanceList
	if err := s.k8sClient.List(ctx, &instances); err != nil {
		l.Error(err, "Failed to list EC2Instances")
	} else {
		for _, inst := range instances.Items {
			if inst.Status.InstanceID != "" {
				// Avoid duplicates
				found := false
				for _, id := range instanceIDs {
					if id == inst.Status.InstanceID {
						found = true
						break
					}
				}
				if !found {
					instanceIDs = append(instanceIDs, inst.Status.InstanceID)
					// Try to get region from spec or status (assuming us-east-1 fallback)
					region := inst.Spec.Region
					if region == "" {
						region = "us-east-1"
					}
					instanceRegionMap[inst.Status.InstanceID] = region
				}
			}
		}
	}

	if len(instanceIDs) == 0 {
		return
	}

	// 2. Fetch costs from OpenCost
	costURL := fmt.Sprintf("%s/allocation?aggregate=node&window=1d", s.opencostURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, costURL, nil)
	costByNode := make(map[string]float64)

	if err == nil {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			l.Error(err, "Failed to fetch OpenCost data, using fallback/empty")
		} else {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				var result struct {
					Code int `json:"code"`
					Data []map[string]struct {
						Name      string  `json:"name"`
						TotalCost float64 `json:"totalCost"`
					} `json:"data"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
					for _, window := range result.Data {
						for nodeName, alloc := range window {
							// OpenCost might return cluster-name/node-name or just node-name
							cleanNodeName := nodeName
							if idx := strings.LastIndex(nodeName, "/"); idx != -1 {
								cleanNodeName = nodeName[idx+1:]
							}
							costByNode[cleanNodeName] = alloc.TotalCost
						}
					}
				} else {
					l.Error(err, "Failed to decode OpenCost response")
				}
			} else {
				l.Info("OpenCost returned non-200 status", "status", resp.StatusCode)
			}
		}
	} else {
		l.Error(err, "Failed to create OpenCost request")
	}

	// 3. AWS Enrichment via DescribeInstances (Group by Region)
	regionToInstances := make(map[string][]string)
	for id, region := range instanceRegionMap {
		regionToInstances[region] = append(regionToInstances[region], id)
	}

	awsEnriched := make(map[string]ec2types.Instance) // instanceId -> instance data
	var wg sync.WaitGroup
	var mu sync.Mutex

	for r, ids := range regionToInstances {
		wg.Add(1)
		go func(region string, instanceIds []string) {
			defer wg.Done()

			cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
			if err != nil {
				l.Error(err, "Failed to load AWS config", "region", region)
				return
			}
			ec2Client := ec2.NewFromConfig(cfg)

			var batchIds []string
			for i, id := range instanceIds {
				batchIds = append(batchIds, id)
				if len(batchIds) == 50 || i == len(instanceIds)-1 {
					describeOutput, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
						InstanceIds: batchIds,
					})
					if err != nil {
						l.Error(err, "DescribeInstances failed", "region", region)
					} else {
						mu.Lock()
						for _, res := range describeOutput.Reservations {
							for _, inst := range res.Instances {
								if inst.InstanceId != nil {
									awsEnriched[*inst.InstanceId] = inst
								}
							}
						}
						mu.Unlock()
					}
					batchIds = nil
				}
			}
		}(r, ids)
	}
	wg.Wait()

	// 4. Combine and Cache Data
	instanceToNode := make(map[string]string)
	for name, n := range nodeMap {
		id, _ := extractInstanceDetails(n.Spec.ProviderID)
		if id != "" {
			instanceToNode[id] = name
		}
	}

	for _, id := range instanceIDs {
		nodeName := instanceToNode[id]
		dailyCost := 0.0
		if nodeName != "" {
			dailyCost = costByNode[nodeName]
		}

		instData := awsEnriched[id]
		instType := ""
		state := ""
		if instData.InstanceType != "" {
			instType = string(instData.InstanceType)
		}
		if instData.State != nil {
			state = string(instData.State.Name)
		}

		// If OpenCost didn't return a cost yet (first sync, node not tracked),
		// fall back to the static on-demand price table so the UI is never blank.
		if dailyCost == 0 && instType != "" {
			dailyCost = dailyCostFromInstanceType(instType)
		}

		costInfo := InstanceCostData{
			InstanceID:   id,
			Node:         nodeName,
			InstanceType: instType,
			Region:       instanceRegionMap[id],
			DailyCost:    dailyCost,
			MonthlyCost:  dailyCost * 30,
			State:        state,
		}

		// Update Prometheus metrics
		// Try to find the instance name from the CRD list
		instanceName := ""
		instanceNamespace := ""
		for _, inst := range instances.Items {
			if inst.Status.InstanceID == id {
				instanceName = inst.Name
				instanceNamespace = inst.Namespace
				break
			}
		}

		labels := prometheus.Labels{
			"instance_id":   id,
			"instance_type": instType,
			"region":        instanceRegionMap[id],
			"namespace":     instanceNamespace,
			"instance_name": instanceName,
		}
		instanceHourlyCost.With(labels).Set(dailyCost / 24)
		instanceCumulativeCost.With(labels).Set(dailyCost * 30)

		s.cache.Store(id, costInfo)
	}
	l.Info("Successfully synced cost data", "count", len(instanceIDs))
}

// GetAllCosts returns all cached cost data
func (s *CostService) GetAllCosts() []InstanceCostData {
	var result []InstanceCostData
	s.cache.Range(func(key, value interface{}) bool {
		result = append(result, value.(InstanceCostData))
		return true
	})
	return result
}

// GetCost returns the cost for a single instanceID
func (s *CostService) GetCost(instanceID string) *InstanceCostData {
	val, ok := s.cache.Load(instanceID)
	if !ok {
		return nil
	}
	data := val.(InstanceCostData)
	return &data
}

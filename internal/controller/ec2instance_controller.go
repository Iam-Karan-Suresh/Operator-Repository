/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"time"

	computev1 "github.com/Iam-Karan-Suresh/operator-repo/api/v1"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	StateRunning    = "running"
	StateTerminated = "terminated"
)

var (
	managedInstances = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "ec2_operator_managed_instances_total",
			Help: "Total number of EC2 instances managed by the operator",
		},
	)
	ReconciliationTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ec2_operator_reconciliation_total",
			Help: "Total number of reconciliation attempts",
		},
	)
	ApiLatency = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ec2_operator_api_latency_seconds",
			Help:    "Latency of AWS API calls",
			Buckets: prometheus.DefBuckets,
		},
	)
	instanceStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ec2_operator_managed_instances_total_by_location",
			Help: "Total number of managed EC2 instances by namespace and region",
		},
		[]string{"namespace", "region"},
	)
	instanceInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ec2_operator_instance_info",
			Help: "Metadata about managed EC2 instances",
		},
		[]string{"instance_id", "instance_name", "namespace", "instance_type", "region"},
	)
	instanceState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ec2_operator_instance_state",
			Help: "Current state of the EC2 instance (0:pending, 1:running, 2:shutting-down, 3:terminated, 4:stopping, 5:stopped)",
		},
		[]string{"instance_id"},
	)
	instanceIPs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ec2_operator_instance_ips",
			Help: "IP addresses of the EC2 instance",
		},
		[]string{"instance_id", "type", "ip_address"},
	)
	instanceProvisionTime = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ec2_operator_instance_provision_duration_seconds",
			Help:    "Time taken to provision an EC2 instance",
			Buckets: []float64{30, 60, 120, 180, 240, 300, 600},
		},
	)
	stateCodes = map[string]float64{
		"pending":       0,
		"running":       1,
		"shutting-down": 2,
		"terminated":    3,
		"stopping":      4,
		"stopped":       5,
	}
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(managedInstances, ReconciliationTotal, ApiLatency, instanceStatus, instanceInfo, instanceProvisionTime, instanceState, instanceIPs)
}

// Ec2InstanceReconciler reconciles an Ec2Instance object.
// The Reconciler is the core component of the operator pattern. It acts as an endless control loop
// ensuring the actual state of the system matches the desired state described in the Ec2Instance YAML.
type Ec2InstanceReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=compute.cloud.com,resources=ec2instances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=compute.cloud.com,resources=ec2instances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=compute.cloud.com,resources=ec2instances/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// This function gets called every time a relevant event occurs:
// - A user applies a new Ec2Instance YAML.
// - A user updates an existing Ec2Instance YAML.
// - A user deletes an Ec2Instance.
// - The periodic resync interval hits.
func (r *Ec2InstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	tracer := otel.GetTracerProvider().Tracer("ec2-operator")
	ctx, span := tracer.Start(ctx, "Reconcile", trace.WithAttributes(
		attribute.String("instance.name", req.Name),
		attribute.String("instance.namespace", req.Namespace),
	))
	defer span.End()

	log.Info("=== RECONCILE LOOP STARTED ===", "namespace", req.Namespace, "name", req.Name)
	ReconciliationTotal.Inc()

	ec2Instance := &computev1.Ec2Instance{}
	if err := r.Get(ctx, req.NamespacedName, ec2Instance); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Instance resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Ec2Instance")
		return ctrl.Result{}, err
	}

	// -------------------------------------------------------------
	// 1. Handle Deletion (if resource is being deleted by the user)
	// -------------------------------------------------------------
	// If the DeletionTimestamp is set, the resource is pending deletion. We must run finalizers.
	if !ec2Instance.DeletionTimestamp.IsZero() {
		log.Info("Instance is being deleted")
		startTime := time.Now()
		_, err := deleteEc2Instance(ctx, ec2Instance)
		ApiLatency.Observe(time.Since(startTime).Seconds())
		if err != nil {
			log.Error(err, "Failed to delete EC2 instance")
			return ctrl.Result{Requeue: true}, err
		}

		// Cleanup Prometheus series for instanceInfo, instanceState, and instanceIPs prior to removing the finalizer
		r.cleanupInstanceMetrics(ec2Instance)

		controllerutil.RemoveFinalizer(ec2Instance, "ec2instance.compute.cloud.com")
		if err := r.Update(ctx, ec2Instance); err != nil {
			log.Error(err, "Failed to remove finalizer")
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, nil
	}

	// -------------------------------------------------------------
	// 2. Add Finalizer (if missing)
	// -------------------------------------------------------------
	// A Finalizer ensures Kubernetes waits for our logic to finish (like deleting the AWS instance)
	// before it completely removes the object from its database.
	if !controllerutil.ContainsFinalizer(ec2Instance, "ec2instance.compute.cloud.com") {
		controllerutil.AddFinalizer(ec2Instance, "ec2instance.compute.cloud.com")
		if err := r.Update(ctx, ec2Instance); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, nil
	}

	// -------------------------------------------------------------
	// 3. Drift Detection (Checking existing EC2 instances in AWS)
	// -------------------------------------------------------------
	// If instance already exists in status, check its state in AWS.
	// This helps us know if AWS instance was terminated directly from the AWS Console.
	if ec2Instance.Status.InstanceID != "" {
		startTime := time.Now()
		exists, instance, err := checkEC2InstanceExists(ctx, ec2Instance.Status.InstanceID, ec2Instance)
		ApiLatency.Observe(time.Since(startTime).Seconds())
		if err != nil {
			log.Error(err, "Failed to check EC2 instance in AWS")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}

		if !exists {
			log.Info("Instance missing in AWS, marking as terminated", "instanceID", ec2Instance.Status.InstanceID)

			// Cleanup old state and IPs
			instanceState.WithLabelValues(ec2Instance.Status.InstanceID).Set(stateCodes[StateTerminated])
			if ec2Instance.Status.PublicIP != "" {
				instanceIPs.DeleteLabelValues(ec2Instance.Status.InstanceID, "public", ec2Instance.Status.PublicIP)
			}
			if ec2Instance.Status.PrivateIP != "" {
				instanceIPs.DeleteLabelValues(ec2Instance.Status.InstanceID, "private", ec2Instance.Status.PrivateIP)
			}

			ec2Instance.Status.State = StateTerminated
			ec2Instance.Status.PublicIP = ""
			ec2Instance.Status.PublicDNS = ""
			// Only decrement if transitioning from a non-terminated state
			if ec2Instance.Status.State != StateTerminated {
				managedInstances.Dec()
				instanceStatus.WithLabelValues(ec2Instance.Namespace, ec2Instance.Spec.Region).Dec()
			}
			if err := r.Status().Update(ctx, ec2Instance); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}

		// Update status from AWS state
		newState := string(instance.State.Name)
		newIP := ""
		if instance.PublicIpAddress != nil {
			newIP = *instance.PublicIpAddress
		}
		newDNS := ""
		if instance.PublicDnsName != nil {
			newDNS = *instance.PublicDnsName
		}
		newPrivIP := ""
		if instance.PrivateIpAddress != nil {
			newPrivIP = *instance.PrivateIpAddress
		}
		newPrivDNS := ""
		if instance.PrivateDnsName != nil {
			newPrivDNS = *instance.PrivateDnsName
		}

		if ec2Instance.Status.State != newState || ec2Instance.Status.PublicIP != newIP {
			log.Info("Drift detected, updating status", "oldState", ec2Instance.Status.State, "newState", newState)

			// Update metrics if state changed to/from running
			if ec2Instance.Status.State != StateRunning && newState == StateRunning {
				managedInstances.Inc()

				// If we have a provisioning start time, observe the duration
				if ec2Instance.Status.ProvisioningStartTime != nil {
					instanceProvisionTime.Observe(time.Since(ec2Instance.Status.ProvisioningStartTime.Time).Seconds())
					ec2Instance.Status.ProvisioningStartTime = nil
				}
			} else if ec2Instance.Status.State == StateRunning && newState != StateRunning {
				managedInstances.Dec()
			}

			ec2Instance.Status.State = newState
			ec2Instance.Status.PublicIP = newIP
			ec2Instance.Status.PublicDNS = newDNS
			ec2Instance.Status.PrivateIP = newPrivIP
			ec2Instance.Status.PrivateDNS = newPrivDNS

			if err := r.Status().Update(ctx, ec2Instance); err != nil {
				log.Error(err, "Failed to update Ec2Instance status")
				return ctrl.Result{}, err
			}
		}

		// Update metrics
		switch newState {
		case StateRunning:
			log.Info("Instance reached running state", "name", ec2Instance.Name)
			r.Recorder.Event(ec2Instance, corev1.EventTypeNormal, "Running", "EC2 Instance is now running")
		case StateTerminated:
			log.Info("Instance reached terminated state", "name", ec2Instance.Name)
			r.Recorder.Event(ec2Instance, corev1.EventTypeNormal, "Terminated", "EC2 Instance has been terminated")
		}

		// Update Prometheus metrics
		// instanceStatus doesn't need to be updated during drift check unless it's a total count change
		// which is handled during creation/deletion.

		// Update info metric (stable labels)
		instanceInfo.WithLabelValues(
			ec2Instance.Status.InstanceID,
			ec2Instance.Name,
			ec2Instance.Namespace,
			ec2Instance.Spec.InstanceType,
			ec2Instance.Spec.Region,
		).Set(1)

		// Update mutable fields in separate metrics
		instanceState.WithLabelValues(ec2Instance.Status.InstanceID).Set(stateCodes[newState])

		// Update IPs with cleanup of old volatile labels
		if ec2Instance.Status.PublicIP != "" && ec2Instance.Status.PublicIP != newIP {
			instanceIPs.DeleteLabelValues(ec2Instance.Status.InstanceID, "public", ec2Instance.Status.PublicIP)
		}
		if newIP != "" {
			instanceIPs.WithLabelValues(ec2Instance.Status.InstanceID, "public", newIP).Set(1)
		}

		if ec2Instance.Status.PrivateIP != "" && ec2Instance.Status.PrivateIP != newPrivIP {
			instanceIPs.DeleteLabelValues(ec2Instance.Status.InstanceID, "private", ec2Instance.Status.PrivateIP)
		}
		if newPrivIP != "" {
			instanceIPs.WithLabelValues(ec2Instance.Status.InstanceID, "private", newPrivIP).Set(1)
		}

		// Periodic resync for drift detection
		if newState != "terminated" {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		return ctrl.Result{}, nil
	}

	// -------------------------------------------------------------
	// 4. Create new instance in AWS
	// -------------------------------------------------------------
	// If we reach here, we know the instance does not exist in AWS yet.
	log.Info("Creating new EC2 Instance in AWS", "name", ec2Instance.Name)
	startTime := time.Now()
	// Persist the provisioning start time
	ec2Instance.Status.ProvisioningStartTime = &metav1.Time{Time: startTime}

	createdInfo, err := createEc2Instance(ctx, ec2Instance)
	ApiLatency.Observe(time.Since(startTime).Seconds())
	if err != nil {
		log.Error(err, "Failed to create EC2 Instance")
		r.Recorder.Event(ec2Instance, corev1.EventTypeWarning, "CreationFailed", err.Error())
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	ec2Instance.Status.InstanceID = createdInfo.InstanceID
	ec2Instance.Status.State = createdInfo.State
	ec2Instance.Status.PublicIP = createdInfo.PublicIP
	ec2Instance.Status.PrivateIP = createdInfo.PrivateIP
	ec2Instance.Status.PublicDNS = createdInfo.PublicDNS
	ec2Instance.Status.PrivateDNS = createdInfo.PrivateDNS

	if err := r.Status().Update(ctx, ec2Instance); err != nil {
		log.Error(err, "Failed to update status after creation")
		return ctrl.Result{}, err
	}

	log.Info("Successfully created instance and updated status", "instanceID", createdInfo.InstanceID)
	// If it's already running (observed immediately after RunInstances)
	if createdInfo.State == StateRunning {
		managedInstances.Inc()
		if ec2Instance.Status.ProvisioningStartTime != nil {
			instanceProvisionTime.Observe(time.Since(ec2Instance.Status.ProvisioningStartTime.Time).Seconds())
			ec2Instance.Status.ProvisioningStartTime = nil

			// Update status again to clear the provisioning start time if it was set
			if err := r.Status().Update(ctx, ec2Instance); err != nil {
				log.Error(err, "Failed to clear provisioning start time")
				return ctrl.Result{}, err
			}
		}
	}

	// Update Prometheus metrics
	instanceStatus.WithLabelValues(ec2Instance.Namespace, ec2Instance.Spec.Region).Inc()
	instanceInfo.WithLabelValues(
		createdInfo.InstanceID,
		ec2Instance.Name,
		ec2Instance.Namespace,
		ec2Instance.Spec.InstanceType,
		ec2Instance.Spec.Region,
	).Set(1)

	instanceState.WithLabelValues(createdInfo.InstanceID).Set(stateCodes[createdInfo.State])
	if createdInfo.PublicIP != "" {
		instanceIPs.WithLabelValues(createdInfo.InstanceID, "public", createdInfo.PublicIP).Set(1)
	}
	if createdInfo.PrivateIP != "" {
		instanceIPs.WithLabelValues(createdInfo.InstanceID, "private", createdInfo.PrivateIP).Set(1)
	}

	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Ec2InstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&computev1.Ec2Instance{}).
		Named("ec2instance").
		Complete(r)
}

// cleanupInstanceMetrics removes all series related to a specific instance to prevent stale metrics.
// Since DeletePartialMatch is only available in K8s component-base metrics, we use standard DeleteLabelValues
// with all available labels from the CR and status for standard Prometheus metrics compatibility.
func (r *Ec2InstanceReconciler) cleanupInstanceMetrics(ec2Instance *computev1.Ec2Instance) {
	if ec2Instance.Status.InstanceID == "" {
		return
	}
	instanceID := ec2Instance.Status.InstanceID

	// instanceInfo uses multiple labels (id, name, namespace, type, region)
	instanceInfo.DeleteLabelValues(
		instanceID,
		ec2Instance.Name,
		ec2Instance.Namespace,
		ec2Instance.Spec.InstanceType,
		ec2Instance.Spec.Region,
	)

	// instanceState only uses instance_id
	instanceState.DeleteLabelValues(instanceID)

	// instanceIPs uses variable IP labels, we clean up what was registered in status
	if ec2Instance.Status.PublicIP != "" {
		instanceIPs.DeleteLabelValues(instanceID, "public", ec2Instance.Status.PublicIP)
	}
	if ec2Instance.Status.PrivateIP != "" {
		instanceIPs.DeleteLabelValues(instanceID, "private", ec2Instance.Status.PrivateIP)
	}
}

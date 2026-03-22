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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"github.com/prometheus/client_golang/prometheus"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
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
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(managedInstances, ReconciliationTotal)
}

// Ec2InstanceReconciler reconciles a Ec2Instance object
type Ec2InstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=compute.cloud.com,resources=ec2instances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=compute.cloud.com,resources=ec2instances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=compute.cloud.com,resources=ec2instances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
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

	// Handle Deletion
	if !ec2Instance.DeletionTimestamp.IsZero() {
		log.Info("Instance is being deleted")
		_, err := deleteEc2Instance(ctx, ec2Instance)
		if err != nil {
			log.Error(err, "Failed to delete EC2 instance")
			return ctrl.Result{Requeue: true}, err
		}

		controllerutil.RemoveFinalizer(ec2Instance, "ec2instance.compute.cloud.com")
		if err := r.Update(ctx, ec2Instance); err != nil {
			log.Error(err, "Failed to remove finalizer")
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, nil
	}

	// Add Finalizer if missing
	if !controllerutil.ContainsFinalizer(ec2Instance, "ec2instance.compute.cloud.com") {
		controllerutil.AddFinalizer(ec2Instance, "ec2instance.compute.cloud.com")
		if err := r.Update(ctx, ec2Instance); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, nil
	}

	// If instance already exists in status, check its state in AWS (Drift Detection)
	if ec2Instance.Status.InstanceID != "" {
		exists, instance, err := checkEC2InstanceExists(ctx, ec2Instance.Status.InstanceID, ec2Instance)
		if err != nil {
			log.Error(err, "Failed to check EC2 instance in AWS")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}

		if !exists {
			log.Info("Instance missing in AWS, marking as terminated", "instanceID", ec2Instance.Status.InstanceID)
			ec2Instance.Status.State = "terminated"
			ec2Instance.Status.PublicIP = ""
			ec2Instance.Status.PublicDNS = ""
			managedInstances.Dec()
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
			if ec2Instance.Status.State != "running" && newState == "running" {
				managedInstances.Inc()
			} else if ec2Instance.Status.State == "running" && newState != "running" {
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
		if newState == "running" {
			managedInstances.Inc()
		} else if newState == "terminated" {
			managedInstances.Dec()
		}

		// Periodic resync for drift detection
		if newState != "terminated" {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		return ctrl.Result{}, nil
	}

	// Create new instance
	log.Info("Creating new EC2 Instance in AWS", "name", ec2Instance.Name)
	createdInfo, err := createEc2Instance(ctx, ec2Instance)
	if err != nil {
		log.Error(err, "Failed to create EC2 Instance")
		return ctrl.Result{}, err
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
	// If it's already running (unlikely immediately, but for consistency)
	if createdInfo.State == "running" {
		managedInstances.Inc()
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

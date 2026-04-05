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
	"github.com/Iam-Karan-Suresh/operator-repo/internal/cache"
	"github.com/Iam-Karan-Suresh/operator-repo/internal/events"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
)

func init() {
	metrics.Registry.MustRegister(managedInstances, ReconciliationTotal, ApiLatency)
}

type Ec2InstanceReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Cache    *cache.InstanceCache // Redis cache
	Events   *events.Producer     // Kafka event producer
}

// +kubebuilder:rbac:groups=compute.cloud.com,resources=ec2instances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=compute.cloud.com,resources=ec2instances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=compute.cloud.com,resources=ec2instances/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;patch

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

	if !ec2Instance.DeletionTimestamp.IsZero() {
		log.Info("Instance is being deleted")
		startTime := time.Now()
		_, err := deleteEc2Instance(ctx, ec2Instance)
		ApiLatency.Observe(time.Since(startTime).Seconds())
		if err != nil {
			log.Error(err, "Failed to delete EC2 instance")
			return ctrl.Result{Requeue: true}, err
		}

		if r.Cache != nil {
			_ = r.Cache.DeleteInstance(ctx, ec2Instance.Status.InstanceID)
		}
		if r.Events != nil {
			_ = r.Events.Publish(ctx, events.InstanceEvent{
				Type:       events.InstanceDeleted,
				InstanceID: ec2Instance.Status.InstanceID,
				Name:       ec2Instance.Name,
				Namespace:  ec2Instance.Namespace,
				State:      StateTerminated,
				Region:     ec2Instance.Spec.Region,
			})
		}

		controllerutil.RemoveFinalizer(ec2Instance, "ec2instance.compute.cloud.com")
		if err := r.Update(ctx, ec2Instance); err != nil {
			log.Error(err, "Failed to remove finalizer")
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(ec2Instance, "ec2instance.compute.cloud.com") {
		controllerutil.AddFinalizer(ec2Instance, "ec2instance.compute.cloud.com")
		if err := r.Update(ctx, ec2Instance); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, nil
	}

	if ec2Instance.Status.InstanceID != "" {
		if r.Cache != nil {
			cached := r.Cache.GetInstanceState(ctx, ec2Instance.Status.InstanceID)
			if cached != nil && cached.State == ec2Instance.Status.State {
				log.V(1).Info("Cache hit, skipping AWS API call", "instanceID", ec2Instance.Status.InstanceID)
				if cached.State != StateTerminated {
					return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
				}
				return ctrl.Result{}, nil
			}
		}

		startTime := time.Now()
		exists, instance, err := checkEC2InstanceExists(ctx, ec2Instance.Status.InstanceID, ec2Instance)
		ApiLatency.Observe(time.Since(startTime).Seconds())
		if err != nil {
			log.Error(err, "Failed to check EC2 instance in AWS")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}

		if !exists {
			log.Info("Instance missing in AWS, marking as terminated", "instanceID", ec2Instance.Status.InstanceID)
			ec2Instance.Status.State = StateTerminated
			ec2Instance.Status.PublicIP = ""
			ec2Instance.Status.PublicDNS = ""
			managedInstances.Dec()

			if r.Cache != nil {
				_ = r.Cache.SetInstanceState(ctx, &cache.CachedInstanceState{
					InstanceID: ec2Instance.Status.InstanceID,
					State:      StateTerminated,
					Region:     ec2Instance.Spec.Region,
				})
			}
			if r.Events != nil {
				_ = r.Events.Publish(ctx, events.InstanceEvent{Type: events.DriftDetected, InstanceID: ec2Instance.Status.InstanceID, Name: ec2Instance.Name, Namespace: ec2Instance.Namespace, State: StateTerminated})
			}

			if err := r.Status().Update(ctx, ec2Instance); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}

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

		if r.Cache != nil {
			_ = r.Cache.SetInstanceState(ctx, &cache.CachedInstanceState{
				InstanceID: ec2Instance.Status.InstanceID,
				State:      newState,
				PublicIP:   newIP,
				PrivateIP:  newPrivIP,
				PublicDNS:  newDNS,
				PrivateDNS: newPrivDNS,
				Region:     ec2Instance.Spec.Region,
			})
		}

		if ec2Instance.Status.State != newState || ec2Instance.Status.PublicIP != newIP {
			log.Info("Drift detected, updating status", "oldState", ec2Instance.Status.State, "newState", newState)
			if r.Events != nil {
				_ = r.Events.Publish(ctx, events.InstanceEvent{Type: events.DriftDetected, InstanceID: ec2Instance.Status.InstanceID, Name: ec2Instance.Name, Namespace: ec2Instance.Namespace, State: newState, PublicIP: newIP, PrivateIP: newPrivIP, Region: ec2Instance.Spec.Region})
			}

			if ec2Instance.Status.State != StateRunning && newState == StateRunning {
				managedInstances.Inc()
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

		switch newState {
		case StateRunning:
			r.Recorder.Event(ec2Instance, corev1.EventTypeNormal, "Running", "EC2 Instance is now running")
		case StateTerminated:
			r.Recorder.Event(ec2Instance, corev1.EventTypeNormal, "Terminated", "EC2 Instance has been terminated")
		}

		if newState != "terminated" {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		return ctrl.Result{}, nil
	}

	log.Info("Creating new EC2 Instance in AWS", "name", ec2Instance.Name)
	startTime := time.Now()
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

	if r.Cache != nil {
		_ = r.Cache.SetInstanceState(ctx, &cache.CachedInstanceState{
			InstanceID: createdInfo.InstanceID,
			State:      createdInfo.State,
			PublicIP:   createdInfo.PublicIP,
			PrivateIP:  createdInfo.PrivateIP,
			PublicDNS:  createdInfo.PublicDNS,
			PrivateDNS: createdInfo.PrivateDNS,
			Region:     ec2Instance.Spec.Region,
		})
	}
	if r.Events != nil {
		_ = r.Events.Publish(ctx, events.InstanceEvent{Type: events.InstanceCreated, InstanceID: createdInfo.InstanceID, Name: ec2Instance.Name, Namespace: ec2Instance.Namespace, State: createdInfo.State, PublicIP: createdInfo.PublicIP, PrivateIP: createdInfo.PrivateIP, Region: ec2Instance.Spec.Region})
	}

	log.Info("Successfully created instance and updated status", "instanceID", createdInfo.InstanceID)
	if createdInfo.State == "running" {
		managedInstances.Inc()
	}
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *Ec2InstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&computev1.Ec2Instance{}).
		Named("ec2instance").
		Complete(r)
}

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

	computev1 "github.com/Iam-Karan-Suresh/operator-repo/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

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
//
// After updating the status of the resource (e.g., with r.Status().Update), the Kubernetes API server
// will emit an update event for the resource. This event will be picked up by the controller-runtime
// and will cause the Reconcile function to be called again for the same resource. This is why, after
// updating the status, the reconciler is called again: it is a result of the Kubernetes watch mechanism
// and ensures that the controller can observe and react to any changes, including those it made itself.
// This pattern is common in Kubernetes controllers to ensure eventual consistency and to handle
// situations where the status update may not have been fully applied or observed yet.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *Ec2InstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := logf.FromContext(ctx)

	l.Info("===RECONCILE LOOP STARTED ===", "namespace", req.Namespace, "name", req.Name)

	// create a new instance of the Ec2Instance struct to hold the data retrieved from the API.
	// This struct will be populated with the current state of the EC2Instance resource specified
	// by the request.
	ec2Instance := &computev1.Ec2Instance{}
	// retrieve the resource from the kubernetes API server using the
	// provided request's Namespace and Name
	if err := r.Get(ctx, req.NamespacedName, ec2Instance); err != nil {
		if errors.IsNotFound(err) {
			l.Info("Instance Deleted. No need to reconcile")
			return ctrl.Result{}, nil
		}
		// kubernetes will retry with backoff
		return ctrl.Result{}, err
	}

	// check if deletionTimestamp is not zero
	if !ec2Instance.DeletionTimestamp.IsZero() {
		l.Info("Has deletionTimestamp, Instance is being deleted")
		_, err := deleteEc2Instance(ctx, ec2Instance)
		if err != nil {
			l.Error(err, "Failed to delete EC2 instance")
			// Kubernetes will retry with backoff
			return ctrl.Result{Requeue: true}, err
		}

		// Remove the finalizer
		controllerutil.RemoveFinalizer(ec2Instance, "ec2instance.compute.cloud.com")
		if err := r.Update(ctx, ec2Instance); err != nil {
			l.Error(err, "Failed to remove finalizer")
			// Kubernetes will retry with backoff
			return ctrl.Result{Requeue: true}, err
		}
		// at this point, the instance state is terminated and the finalizer is removed
		return ctrl.Result{}, nil
	}

	// if errors.IsNotFound(err) {
	// 	// Object was deleted
	// 	fmt.Println("Ran kubectl delete ec2instance")
	// 	l.Info("Got a delete request for the instance. Will delete the instance from AWS")
	// 	// Any cleanup logic here (though you can't access the object anymore)
	// 	return ctrl.Result{}, nil
	// }

	// Check if we already have an instance ID in status
	if ec2Instance.Status.InstanceID != "" {
		l.Info("Requested object is already exists in kubernetes. Not creating a new instance", "instanceID", ec2Instance.Status.InstanceID)
		
		//drift detection mechanism
		instanceExist, instanceState, err := checkEC2InstanceExists(ctx, ec2Instance.Status.InstanceID, ec2Instance)
		if err != nil {
			ec2Instance.Status.InstanceID = ""
			ec2Instance.Status.State = ""
			ec2Instance.Status.PublicIP = ""
			ec2Instance.Status.PrivateIP = ""
			ec2Instance.Status.PublicDNS = ""
			ec2Instance.Status.PrivateDNS = ""
			return ctrl.Result{Requeue: true}, r.Status().Update(ctx, ec2Instance)
		}
		if !instanceExist {
			l.Info("Instance does not exist or is not running", "instanceID", ec2Instance.Status.InstanceID)
			ec2Instance.Status.State = "Unknown"
			ec2Instance.Status.PublicIP = ""
			r.Status().Update(ctx, ec2Instance)
			return ctrl.Result{}, nil
		}
		if instanceExist && ec2Instance.Status.State == "Unknown" {
			l.Info("Found a running Instance", "instanceID", ec2Instance.Status.InstanceID)
			ec2Instance.Status.State = string(*instanceState.InstanceId)
			ec2Instance.Status.PublicIP = *instanceState.PublicIpAddress
			r.Status().Update(ctx, ec2Instance)
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, nil
	}

	l.Info("Creating new Instance")
	// Check if already contains finalizer, add it otherwise
	if !controllerutil.ContainsFinalizer(ec2Instance, "ec2instance.compute.cloud.com") {
		l.Info(" === ABOUT TO ADD FINALIZER ===")
		controllerutil.AddFinalizer(ec2Instance, "ec2instance.compute.cloud.com")
		if err := r.Update(ctx, ec2Instance); err != nil {
			l.Error(err, "Failed to add finalizer")
			return ctrl.Result{Requeue: true}, err
		}
		l.Info(" === FINALIZER ADDED - Returning to trigger new reconcile loop cleanly ===")
		return ctrl.Result{}, nil
	}

	// create a new Instance
	l.Info(" === CONTINUING WITH EC2 INSTANCE CREATION IN CURRENT RECONCILE LOOP ===")

	createdInstanceInfo, err := createEc2Instance(ec2Instance)
	if err != nil {
		l.Error(err, "Failed to create EC2 Instance")
		return ctrl.Result{}, err
	}

	l.Info("=== ABOUT TO UPDATE STATUS - This will trigger reconciler loop again ===",
		"instanceID", createdInstanceInfo.InstanceID,
		"state", createdInstanceInfo.State)

	ec2Instance.Status.InstanceID = createdInstanceInfo.InstanceID
	ec2Instance.Status.State = createdInstanceInfo.State
	ec2Instance.Status.PublicIP = createdInstanceInfo.PublicIP
	ec2Instance.Status.PrivateIP = createdInstanceInfo.PrivateIP
	ec2Instance.Status.PublicDNS = createdInstanceInfo.PublicDNS
	ec2Instance.Status.PrivateDNS = createdInstanceInfo.PrivateDNS

	err = r.Status().Update(ctx, ec2Instance)
	if err != nil {
		l.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}
	l.Info(" === STATUS UPDATED - Reconcile loop will be triggered again ===")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Ec2InstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&computev1.Ec2Instance{}).
		Named("ec2instance").
		Complete(r)
}

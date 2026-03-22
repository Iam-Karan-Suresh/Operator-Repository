package controller

import (
	"context"
	"time"

	computev1 "github.com/Iam-Karan-Suresh/operator-repo/api/v1"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func deleteEc2Instance(ctx context.Context, ec2Instance *computev1.Ec2Instance) (bool, error) {
	log := logf.FromContext(ctx).WithName("deleteEc2Instance")

	tracer := otel.GetTracerProvider().Tracer("ec2-operator")
	ctx, span := tracer.Start(ctx, "AWS.TerminateInstance", trace.WithAttributes(
		attribute.String("instance.id", ec2Instance.Status.InstanceID),
	))
	defer span.End()

	log.Info("Deleting EC2 instance", "instanceID", ec2Instance.Status.InstanceID)

	ec2Client := awsClient(ec2Instance.Spec.Region)

	// Terminate the instance
	terminateResult, err := ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{ec2Instance.Status.InstanceID},
	})

	if err != nil {
		log.Error(err, "Failed to terminate EC2 instance")
		return false, err
	}

	if len(terminateResult.TerminatingInstances) > 0 {
		log.Info("Instance termination initiated",
			"instanceID", ec2Instance.Status.InstanceID,
			"currentState", terminateResult.TerminatingInstances[0].CurrentState.Name)
	}

	waiter := ec2.NewInstanceTerminatedWaiter(ec2Client)
	maxWaitTime := 5 * time.Minute

	waitCtx, waitCancel := context.WithTimeout(ctx, maxWaitTime)
	defer waitCancel()

	log.Info("Waiting for instance to be terminated",
		"instanceID", ec2Instance.Status.InstanceID,
		"maxWaitTime", maxWaitTime)

	err = waiter.Wait(waitCtx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{ec2Instance.Status.InstanceID},
	}, maxWaitTime)

	if err != nil {
		log.Error(err, "Failed while waiting for instance termination",
			"instanceID", ec2Instance.Status.InstanceID)
		return false, err
	}

	log.Info("EC2 instance successfully terminated", "instanceID", ec2Instance.Status.InstanceID)
	return true, nil
}

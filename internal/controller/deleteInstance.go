package controller

import (
	"context"

	computev1 "github.com/Iam-Karan-Suresh/operator-repo/api/v1"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func deleteEc2Instance(ctx context.Context, ec2Instance *computev1.Ec2Instance) (bool, error) {
	l := log.FromContext(ctx)

	l.Info("Deleting EC2 instance", "instanceID", ec2Instance.Status.InstanceID)

	// create the client for ec2 instance
	ec2Client := awsClient(ec2Instance.Spec.Region)

	if ec2Instance.Status.InstanceID == "" {
		l.Info("InstanceID is empty. Nothing to delete on AWS")
		return true, nil
	}

	// Terminate the instance
	terminateResult, err := ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{ec2Instance.Status.InstanceID},
	})

	if err != nil {
		l.Error(err, "Failed to terminate EC2 instance")
		return false, err
	}

	l.Info("Instance termination initiated",
		"instanceID", ec2Instance.Status.InstanceID,
		"currentState", terminateResult.TerminatingInstances[0].CurrentState.Name)

	l.Info("EC2 instance termination initiated successfully via AWS API. Not blocking for final termination state.", "instanceID", ec2Instance.Status.InstanceID)
	return true, nil
}

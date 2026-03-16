package controller

import (
	"context"
	"time"

	computev1 "github.com/Iam-Karan-Suresh/operator-repo/api/v1"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func deleteEc2Instance(ctx context.Context, ec2Instance *computev1.Ec2Instance) (bool, error) {
	l := log.FromContext(ctx)

	l.Info("Deleting the instance", "instanceID", ec2Instance.Status.InstanceID)

	//create ec2 instance client using awsClient 
	ec2Client := awsClient(ec2Instance.Spec.Region)

	// Terminate ec2 instance 
	terminateResult, err := ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{ec2Instance.Status.InstanceID},
	})	
	if err != nil {
		l.Error(err, "Failed to delete EC2 instance")
		return false, err
	}
	l.Info("Instance termination initiated",
                "instanceID", ec2Instance.Status.InstanceID,
                "currentState", terminateResult.TerminatingInstances[0].CurrentState.Name)

	// Use the AWS SDK v2 waiter to efficiently wait for instance termination 
	// The waiter uses exponential backoff and is more efficient than manual polling

	waiter := ec2.NewInstanceTerminatedWaiter(ec2Client)
	
	maxWaitTime := 5 * time.Minute //maximum wait time to terminate instances 


	// DescribeInstancesInput is to define the criteria for querying information about your Amazon EC2 instances. mainy used for quering the details of specified instance.
	waitParams := &ec2.DescribeInstancesInput{
		InstanceIds: []string{ec2Instance.Status.InstanceID},

	}

	// waiting for the instance to be terminated
	err = waiter.Wait(ctx, waitParams, maxWaitTime)

	if err != nil {
		l.Error(err, "Failed while waiting for instance termination", "instanceID", ec2Instance.Status.InstanceID,
	"macWaitTime", maxWaitTime)
		return false, err
	}
	l.Info("EC2 instance  successfully terminated", "instanceID", ec2Instance.Status.InstanceID)
	return true, nil
	
}

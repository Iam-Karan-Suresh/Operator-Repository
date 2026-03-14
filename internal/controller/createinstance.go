package controller

import (
	"context"
	"fmt"
	"time"

	computev1 "github.com/Iam-Karan-Suresh/operator-repo/api/v1"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func createEc2Instance(ec2Instance *computev1.Ec2Instance) (createdInstanceInfo *computev1.CreatedInstanceInfo, err error) {
	l := log.Log.WithName("createEc2Instance")

	l.Info(" === STARTING EC2 INSTANCE CREATION PROCESS === ",
		"ami", ec2Instance.Spec.AMIId,
		"instanceType", ec2Instance.Spec.InstanceType,
		"region", ec2Instance.Spec.Region,
	)
	//create the client for the ec2 instance
	ec2Client := awsClient(ec2Instance.Spec.Region)

	//create the input for the ec2 instance
	runInput := &ec2.RunInstancesInput{
		ImageId:      aws.String(ec2Instance.Spec.AMIId),
		InstanceType: ec2types.InstanceType(ec2Instance.Spec.InstanceType),
		SubnetId:     aws.String(ec2Instance.Spec.Subnet),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		//SecurityGroupIds: []string{ec2Instance.Spec.SecurityGroup[0]},
	}
	l.Info(" === CALLING AWS RunInstances API === ")
	//run the instances
	result, err := ec2Client.RunInstances(context.TODO(), runInput)
	if err != nil {
		l.Error(err, "failed to create EC2 instance: %w", err)
	}
	if len(result.Instances) == 0 {
		l.Error(err, "no instance returned in RunInstancesOutput")
		fmt.Println("No instances returned in RunInstancesOutput")
		return nil, nil
	}

	//till here the instance is created and we have
	//Instance ID, private dns and IP, instance type,image id
	inst := result.Instances[0]
	l.Info(" === EC2 INSTANCE CREATED SUCCESSFULLY === ", "instanceID", *inst.InstanceId)
	l.Info(" === WAITING FOR INSTANCE TO BE IN RUNNING STATE === ")
	runWaiter := ec2.NewInstanceRunningWaiter(ec2Client)
	maxWaitTime := 3 * time.Minute

	err = runWaiter.Wait(context.TODO(), &ec2.DescribeInstancesInput{
		InstanceIds: []string{*inst.InstanceId},
	}, maxWaitTime)
	if err != nil {
		l.Error(err, "failed to wait for instance to be in running state")
		return nil, fmt.Errorf("failed to wait for instance to be in running state: %w", err)
	}

	// After creating the instance, we waited and now we describe to
	// 1.Get the public IP and dns as it takes some time for it
	// 2. Getting the state of the instance
	// we do this so we can send the instance's state to the status of the custom resource. for user to see with
	l.Info(" === CALLING AWS DescribeInstances API TO GET INSTACE DETAILS ===")
	describeInput := &ec2.DescribeInstancesInput{
		InstanceIds: []string{*inst.InstanceId},
	}

	describeResult, err := ec2Client.DescribeInstances(context.TODO(), describeInput)

	if err != nil {
		l.Error(err, "Failed to describe the EC2 instance")
		return nil, fmt.Errorf("failed to describe EC2 instance: %w", err)
	}
	fmt.Println("Describe result", "public ip", *describeResult.Reservations[0].Instances[0].PublicDnsName, "state", *&describeResult.Reservations[0].Instances[0].State)

	// you get "invalid memory address or nil pointer dereference here if any of the following are true"
	// - result.Instances is nil or has length 0
	// - any of the pointer fields ( e.g, PublicIpAddress, privateAddress, etc.) are nil

	// To avoid this, always check for nil and length before dereferencing:

	//wait for a bit to allow instance fields to be populated

	fmt.Printf("Private IP of the instance: %v", derefString(inst.PrivateIpAddress))
	fmt.Printf("State of the instance: %v", describeResult.Reservations[0].Instances[0].State.Name)
	fmt.Printf("Private DNS name of the instance: %v", derefString(inst.PrivateDnsName))
	fmt.Printf("InstanceId of the instance: %v", derefString(inst.InstanceId))
	fmt.Printf("Image ID of the instance: %v", derefString(inst.ImageId))
	fmt.Printf("Key name of the instance: %v", derefString(inst.KeyName))

	//block until the instance is  running
	//blockUntilInstanceRunning(ctx, ec2Instance.Status.InstanceID, ec2Instance)

	// Get the instance details safely (public IP/DNS might be nil for private subnets)
	instance := describeResult.Reservations[0].Instances[0]
	createdInstanceInfo = &computev1.CreatedInstanceInfo{
		InstanceID: *inst.InstanceId,
		PublicIP:   derefString(instance.PublicIpAddress),
		State:      string(instance.State.Name),
		PrivateIP:  derefString(instance.PrivateIpAddress),
		PublicDNS:  derefString(instance.PublicDnsName),
		PrivateDNS: derefString(instance.PrivateDnsName),
	}
	l.Info("=== EC2 INSTANCE CREATION COMPLETED ===",
		"instanceID", createdInstanceInfo.InstanceID,
		"state", createdInstanceInfo.State,
		"publicIP", createdInstanceInfo.PublicIP,
	)

	// for now return nil to indicate success
	return createdInstanceInfo, nil

}

// derefString is a helper function to dereference  *string
func derefString(s *string) string {
	if s != nil {
		return *s
	}
	return "<nil>"
}

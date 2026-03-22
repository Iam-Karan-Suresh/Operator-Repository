package controller

import (
	"context"
	"fmt"
	"time"

	computev1 "github.com/Iam-Karan-Suresh/operator-repo/api/v1"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func createEc2Instance(ctx context.Context, ec2Instance *computev1.Ec2Instance) (createdInstanceInfo *computev1.CreatedInstanceInfo, err error) {
	log := logf.FromContext(ctx).WithName("createEc2Instance")

	tracer := otel.GetTracerProvider().Tracer("ec2-operator")
	ctx, span := tracer.Start(ctx, "AWS.CreateInstance", trace.WithAttributes(
		attribute.String("instance.name", ec2Instance.Name),
		attribute.String("instance.type", ec2Instance.Spec.InstanceType),
	))
	defer span.End()

	log.Info(" === STARTING EC2 INSTANCE CREATION PROCESS === ",
		"ami", ec2Instance.Spec.AMIId,
		"instanceType", ec2Instance.Spec.InstanceType,
		"region", ec2Instance.Spec.Region,
	)

	ec2Client := awsClient(ec2Instance.Spec.Region)

	runInput := &ec2.RunInstancesInput{
		ImageId:      aws.String(ec2Instance.Spec.AMIId),
		InstanceType: ec2types.InstanceType(ec2Instance.Spec.InstanceType),
		SubnetId:     aws.String(ec2Instance.Spec.Subnet),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	}

	log.Info(" === CALLING AWS RunInstances API === ")
	result, err := ec2Client.RunInstances(ctx, runInput)
	if err != nil {
		log.Error(err, "failed to create EC2 instance")
		return nil, fmt.Errorf("failed to create EC2 instance: %w", err)
	}

	if len(result.Instances) == 0 {
		return nil, fmt.Errorf("no instances returned in RunInstancesOutput")
	}

	inst := result.Instances[0]
	log.Info(" === EC2 INSTANCE CREATED SUCCESSFULLY === ", "instanceID", *inst.InstanceId)
	log.Info(" === WAITING FOR INSTANCE TO BE IN RUNNING STATE === ")

	runWaiter := ec2.NewInstanceRunningWaiter(ec2Client)
	maxWaitTime := 3 * time.Minute

	waitCtx, waitCancel := context.WithTimeout(ctx, maxWaitTime)
	defer waitCancel()

	err = runWaiter.Wait(waitCtx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{*inst.InstanceId},
	}, maxWaitTime)
	if err != nil {
		log.Error(err, "failed to wait for instance to be in running state")
		return nil, fmt.Errorf("failed to wait for instance to be in running state: %w", err)
	}

	log.Info(" === CALLING AWS DescribeInstances API TO GET INSTANCE DETAILS ===")
	describeInput := &ec2.DescribeInstancesInput{
		InstanceIds: []string{*inst.InstanceId},
	}

	describeResult, err := ec2Client.DescribeInstances(ctx, describeInput)
	if err != nil {
		log.Error(err, "Failed to describe the EC2 instance")
		return nil, fmt.Errorf("failed to describe EC2 instance: %w", err)
	}

	instance := describeResult.Reservations[0].Instances[0]
	createdInstanceInfo = &computev1.CreatedInstanceInfo{
		InstanceID: *inst.InstanceId,
		PublicIP:   derefString(instance.PublicIpAddress),
		State:      string(instance.State.Name),
		PrivateIP:  derefString(instance.PrivateIpAddress),
		PublicDNS:  derefString(instance.PublicDnsName),
		PrivateDNS: derefString(instance.PrivateDnsName),
	}

	log.Info("=== EC2 INSTANCE CREATION COMPLETED ===",
		"instanceID", createdInstanceInfo.InstanceID,
		"state", createdInstanceInfo.State,
		"publicIP", createdInstanceInfo.PublicIP,
	)

	return createdInstanceInfo, nil
}

// derefString is a helper function to dereference *string
func derefString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

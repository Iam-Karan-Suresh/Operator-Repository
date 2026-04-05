// Package controller implements the Kubernetes custom controllers for this operator.
package controller

import (
	"context"
	"encoding/base64"
	"fmt"

	computev1 "github.com/Iam-Karan-Suresh/operator-repo/api/v1"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// createEc2Instance is responsible for translating the Kubernetes Ec2Instance desired state into actual AWS resources.
// It maps our Custom Resource (CR) fields (like InstanceType, AMI ID, Tags, Storage) to the AWS RunInstances API payload.
// It also sets up tracing and initiates the instance creation, returning immediately with initial metadata.
// The instance will be in "pending" state; the controller reconciliation loop handles subsequent state updates.
//
// Returns:
// - CreatedInstanceInfo: A struct containing initial instance metadata (ID, state). Note: IPs and DNS names may be empty until the instance reaches "running" state.
// - err: Any error encountered during the AWS provisioning process.
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

	ec2Client, err := awsClient(ctx, ec2Instance.Spec.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	// Prepare TagSpecifications for Name and other tags
	tagSpecs := []ec2types.TagSpecification{
		{
			ResourceType: ec2types.ResourceTypeInstance,
			Tags: []ec2types.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String(ec2Instance.Name),
				},
			},
		},
	}

	// Add additional tags from spec if any
	for k, v := range ec2Instance.Spec.Tags {
		if k != "Name" { // Don't override our primary Name tag
			tagSpecs[0].Tags = append(tagSpecs[0].Tags, ec2types.Tag{
				Key:   aws.String(k),
				Value: aws.String(v),
			},
			)
		}
	}

	// Storage configuration parsing (Elastic Block Store - EBS).
	// We calculate how many block device mappings we need based on root volume AND additional volumes from the CR.
	cap := 1
	if len(ec2Instance.Spec.Storage.AdditionalVolumes) > 0 {
		cap += len(ec2Instance.Spec.Storage.AdditionalVolumes)
	}
	blockDeviceMappings := make([]ec2types.BlockDeviceMapping, 0, cap)

	// Configure Root volume mapping if specified.
	// Typically attached to /dev/sda1, but this can vary depending on the AMI Linux distro.
	if ec2Instance.Spec.Storage.RootVolume.Size > 0 {
		mapping := ec2types.BlockDeviceMapping{
			DeviceName: aws.String("/dev/sda1"), // Default root device for many AMIs
			Ebs: &ec2types.EbsBlockDevice{
				VolumeSize: aws.Int32(ec2Instance.Spec.Storage.RootVolume.Size),
				VolumeType: ec2types.VolumeType(ec2Instance.Spec.Storage.RootVolume.Type),
				Encrypted:  aws.Bool(ec2Instance.Spec.Storage.RootVolume.Encrypted),
			},
		}
		blockDeviceMappings = append(blockDeviceMappings, mapping)
	}

	// Configure Additional attached volumes if any are specified in the CR spec
	for _, vol := range ec2Instance.Spec.Storage.AdditionalVolumes {
		mapping := ec2types.BlockDeviceMapping{
			DeviceName: aws.String(vol.DeviceName),
			Ebs: &ec2types.EbsBlockDevice{
				VolumeSize: aws.Int32(vol.Size),
				VolumeType: ec2types.VolumeType(vol.Type),
				Encrypted:  aws.Bool(vol.Encrypted),
			},
		}
		blockDeviceMappings = append(blockDeviceMappings, mapping)
	}

	// Construct the primary AWS API payload for launching the instance
	runInput := &ec2.RunInstancesInput{
		ImageId:           aws.String(ec2Instance.Spec.AMIId),
		InstanceType:      ec2types.InstanceType(ec2Instance.Spec.InstanceType),
		SubnetId:          aws.String(ec2Instance.Spec.Subnet),
		MinCount:          aws.Int32(1),
		MaxCount:          aws.Int32(1),
		TagSpecifications: tagSpecs,
	}

	// Add Placement (Availability Zone)
	if ec2Instance.Spec.AvailabilityZone != "" {
		runInput.Placement = &ec2types.Placement{
			AvailabilityZone: aws.String(ec2Instance.Spec.AvailabilityZone),
		}
	}

	// Add KeyPair
	if ec2Instance.Spec.KeyPair != "" {
		runInput.KeyName = aws.String(ec2Instance.Spec.KeyPair)
	}

	// Add Security Groups
	if len(ec2Instance.Spec.SecurityGroups) > 0 {
		runInput.SecurityGroupIds = ec2Instance.Spec.SecurityGroups
	}

	// Add UserData (must be base64 encoded)
	if ec2Instance.Spec.UserData != "" {
		encodedUserData := base64.StdEncoding.EncodeToString([]byte(ec2Instance.Spec.UserData))
		runInput.UserData = aws.String(encodedUserData)
	}

	// Add Storage
	if len(blockDeviceMappings) > 0 {
		runInput.BlockDeviceMappings = blockDeviceMappings
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

	createdInstanceInfo = &computev1.CreatedInstanceInfo{
		InstanceID: *inst.InstanceId,
		State:      string(inst.State.Name),
		PublicIP:   derefString(inst.PublicIpAddress),
		PrivateIP:  derefString(inst.PrivateIpAddress),
		PublicDNS:  derefString(inst.PublicDnsName),
		PrivateDNS: derefString(inst.PrivateDnsName),
	}

	log.Info("=== EC2 INSTANCE CREATION INITIATED ===",
		"instanceID", createdInstanceInfo.InstanceID,
		"state", createdInstanceInfo.State,
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

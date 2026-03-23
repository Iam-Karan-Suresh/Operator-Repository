package controller

import (
	"context"
	"errors"
	"fmt"

	computev1 "github.com/Iam-Karan-Suresh/operator-repo/api/v1"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func checkEC2InstanceExists(ctx context.Context, instanceID string, ec2Instance *computev1.Ec2Instance) (bool, *ec2types.Instance, error) {
	tracer := otel.GetTracerProvider().Tracer("ec2-operator")
	ctx, span := tracer.Start(ctx, "AWS.DescribeInstances", trace.WithAttributes(
		attribute.String("instance.id", instanceID),
	))
	defer span.End()

	ec2Client, err := awsClient(ctx, ec2Instance.Spec.Region)
	if err != nil {
		return false, nil, fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}

	result, err := ec2Client.DescribeInstances(ctx, input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "InvalidInstance.NotFound" {
			return false, nil, nil
		}
		return false, nil, err
	}
	fmt.Println("Length of Reservations are ", len(result.Reservations))

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return false, nil, nil
	}
	return true, &result.Reservations[0].Instances[0], nil

}

// Package controller implements the Kubernetes custom controllers for this operator.
// It contains the logic to watch, evaluate, and manage the lifecycle of AWS resources (like EC2)
// based on the custom resources defined in the Kubernetes cluster.
package controller

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// awsClient initializes and returns a new AWS EC2 Service Client configured for the specified region.
// It loads the default AWS configuration which integrates with the AWS SDK's default credential chain
// (e.g., Environment Variables, ~/.aws/credentials, or IAM Roles context like EKS IRSA).
// This client is then used for all underlying API calls like RunInstances, TerminateInstances, etc.
func awsClient(ctx context.Context, region string) (*ec2.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}
	return ec2.NewFromConfig(cfg), nil
}

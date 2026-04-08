package controller

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var clientPool sync.Map // map[string]*ec2.Client

func awsClient(ctx context.Context, region string) (*ec2.Client, error) {
	if cached, ok := clientPool.Load(region); ok {
		return cached.(*ec2.Client), nil
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	client := ec2.NewFromConfig(cfg)
	actual, _ := clientPool.LoadOrStore(region, client)
	return actual.(*ec2.Client), nil
}

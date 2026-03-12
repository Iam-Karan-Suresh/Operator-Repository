package controller

import (
	"context"
	"fmt"
	"time"	

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	computev1 "github.com/Iam-Karan-Suresh/operator-repo/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func createEc2Instance(ec2Instance *computev1.Ec2Instance) (createdInstanceInfo *computev1.CreatedInstanceInfo, err error) {
	
	
}
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
		ImageId: aws.String(ec2Instance.Spec.AMIId),
		InstanceType: ec2types.InstanceType(ec2Instance.Spec.InstanceType),
		SubnetId: aws.String(ec2Instance.Spec.Subnet),
		MinCount: aws.Int32(1),
		MaxCount: aws.Int32(1),
		//SecurityGroupIds: []string{ec2Instance.Spec.SecurityGroup[0]},
	}
	l.Info(" === CALLING AWS RunInstances API === ")
	//run the instances
	result, err := ec2Client.RunInstances(context.TODO(), runInput)
	if err != nil {
		l.Error(err, "failed to create EC2 instance: %w", err)
	}
	if  len(result.Instances) ==0 {
		l.Error(err, "no instance returned in RunInstancesOutput")
		fmt.Println("No instances returned in RunInstancesOutput")
		return nil, nil
	}

	
}
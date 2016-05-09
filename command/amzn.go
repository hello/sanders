package command

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
)

type AmznServices struct {
	Iam    *iam.IAM
	Ec2    *ec2.EC2
	Elb    *elb.ELB
	Asg    *autoscaling.AutoScaling
	S3     *s3.S3
	S3Keys *s3.S3
}

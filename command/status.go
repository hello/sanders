package command

import (
	"fmt"
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/awslabs/aws-sdk-go/service/elb"
	"github.com/mitchellh/cli"
	"strings"
	// "sync"
)

type StatusCommand struct {
	Ui cli.ColoredUi
}

func (c *StatusCommand) Help() string {
	helpText := `Usage: sanders status`
	return strings.TrimSpace(helpText)
}

func (c *StatusCommand) Run(args []string) int {

	config := &aws.Config{
		Region: "us-east-1",
	}

	service := elb.New(config)
	ec2Service := ec2.New(config)

	for _, elbName := range []string{"suripu-app-prod", "suripu-service-prod"} {
		c.Ui.Info(fmt.Sprintf("ELB: %s", elbName))

		req := &elb.DescribeInstanceHealthInput{
			LoadBalancerName: &elbName,
		}

		lbResp, err := service.DescribeInstanceHealth(req)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("%v", err))
			return 0
		}

		ids := make([]*string, 0)
		// amiIds := make([]string, 0)

		// store := make(map[string]ec2.Instance)
		// mu := sync.Mutex{}
		// statusStore := make(map[string]string)
		// nameStore := make(map[string]string)
		// amiStore := make(map[string]ec2.Image)

		for _, state := range lbResp.InstanceStates {
			c.Ui.Info(fmt.Sprintf("\tID: %s", *state.InstanceID))
			c.Ui.Info(fmt.Sprintf("\tState: %s", *state.State))
			c.Ui.Info(fmt.Sprintf("\tReason: %s", *state.ReasonCode))
			c.Ui.Info(fmt.Sprintf("\tDescription: %s", *state.Description))
			ids = append(ids, state.InstanceID)
		}

		instanceReq := &ec2.DescribeInstancesInput{
			InstanceIDs: ids,
		}

		resp, err := ec2Service.DescribeInstances(instanceReq)
		// instancesResp, err := ecSquare.DescribeInstances(ids, nil)
		// if err != nil {
		// 	c.Ui.Error(fmt.Sprintf("Error: %s", err))
		// 	return 1
		// }

		amiIds := make([]*string, 0)
		for _, reservation := range resp.Reservations {
			for _, instance := range reservation.Instances {
				c.Ui.Info(fmt.Sprintf("%v", *instance.PublicDNSName))
				c.Ui.Info(fmt.Sprintf("%v", *instance.ImageID))
				amiIds = append(amiIds, instance.ImageID)
			}
		}

		amiReq := &ec2.DescribeImagesInput{
			ImageIDs: amiIds,
		}

		amiResp, err := ec2Service.DescribeImages(amiReq)
		for _, image := range amiResp.Images {
			c.Ui.Info(fmt.Sprintf("%s", *image.Name))
		}
		// amiResp, err := ecSquare.Images(amiIds, nil)
		// for _, image := range amiResp.Images {
		// 	amiStore[image.Id] = image
		// }

		// for k, _ := range store {

		// 	msg := fmt.Sprintf("→\t%s\t%s\t%s\t%s\t%s\t%s [%s]",
		// 		statusStore[k],
		// 		nameStore[k],
		// 		store[k].PrivateDNSName,
		// 		store[k].DNSName,
		// 		store[k].InstanceId,
		// 		amiStore[store[k].ImageId].Name,
		// 		amiStore[store[k].ImageId].Description,
		// 	)
		// 	if statusStore[k] == "InService" {
		// 		c.Ui.Info(msg)
		// 	} else {
		// 		c.Ui.Error(msg)
		// 	}
		// }
		// c.Ui.Info("")

		// groupnames := make([]string, 2)
		// groupnames[0] = "suripu-workers-prod"
		// groupnames[1] = "suripu-workers-prod-green"

		// req := &autoscaling.AutoScalingGroupNamesType{
		// 	AutoScalingGroupNames: groupnames,
		// }
		// resp, err := cli.DescribeAutoScalingGroups(req)
	}

	c.Ui.Output("")
	return 0
}

func (c *StatusCommand) Synopsis() string {
	return "See ELB status"
}

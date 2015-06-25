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

	for _, elbName := range []string{"suripu-service-prod", "suripu-app-prod"} {
		c.Ui.Info(fmt.Sprintf("ELB: %s", elbName))

		req := &elb.DescribeInstanceHealthInput{
			LoadBalancerName: &elbName,
		}

		lbResp, err := service.DescribeInstanceHealth(req)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("%v", err))
			return 0
		}

		instanceIds := make([]*string, 0)

		for _, state := range lbResp.InstanceStates {
			instanceIds = append(instanceIds, state.InstanceID)
		}

		instanceReq := &ec2.DescribeInstancesInput{
			InstanceIDs: instanceIds,
		}

		resp, err := ec2Service.DescribeInstances(instanceReq)
		// instancesResp, err := ecSquare.DescribeInstances(ids, nil)
		// if err != nil {
		// 	c.Ui.Error(fmt.Sprintf("Error: %s", err))
		// 	return 1
		// }

		publicNames := make(map[string]string, 0)
		for _, reservation := range resp.Reservations {
			for _, instance := range reservation.Instances {
				publicNames[*instance.InstanceID] = *instance.PublicDNSName
			}
		}

		for _, state := range lbResp.InstanceStates {
			res, ok := publicNames[*state.InstanceID]
			if *state.State == "InService" {
				c.Ui.Info(fmt.Sprintf("\tID: %s", *state.InstanceID))
				c.Ui.Info(fmt.Sprintf("\tState: %s", *state.State))
				if ok {
					c.Ui.Info(fmt.Sprintf("\tHostname: %s", res))
				}

			} else {
				c.Ui.Error(fmt.Sprintf("\tID: %s", *state.InstanceID))
				c.Ui.Error(fmt.Sprintf("\tReason: %s", *state.ReasonCode))
				c.Ui.Error(fmt.Sprintf("\tDescription: %s", *state.Description))
				if ok {
					c.Ui.Error(fmt.Sprintf("\tHostname: %s", res))
				}

			}
			c.Ui.Output("")
		}
	}

	c.Ui.Output("")
	return 0
}

func (c *StatusCommand) Synopsis() string {
	return "See ELB status"
}

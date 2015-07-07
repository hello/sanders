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

	for _, elbName := range []string{"suripu-service-prod", "suripu-app-prod", "suripu-app-canary"} {
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

		resp, _ := ec2Service.DescribeInstances(instanceReq)

		publicNames := make(map[string]string, 0)
		amis := make(map[string]string, 0)
		amisNames := make(map[string]string, 0)
		amisToFetch := make([]*string, 0)
		for _, reservation := range resp.Reservations {
			for _, instance := range reservation.Instances {
				publicNames[*instance.ImageID] = *instance.PublicDNSName
				amis[*instance.InstanceID] = *instance.ImageID
				amisToFetch = append(amisToFetch, instance.ImageID)
			}
		}

		amiReq := &ec2.DescribeImagesInput{
			ImageIDs: amisToFetch,
		}

		amiResp, _ := ec2Service.DescribeImages(amiReq)
		for _, ami := range amiResp.Images {
			amisNames[*ami.ImageID] = *ami.Name
		}

		for _, state := range lbResp.InstanceStates {
			res, ok := publicNames[*state.InstanceID]
			amiId, _ := amis[*state.InstanceID]
			amiName, _ := amisNames[amiId]
			parts := strings.SplitAfterN(amiName, "-", 4)

			if *state.State == "InService" {
				c.Ui.Info(fmt.Sprintf("\tVersion: %s", strings.TrimSuffix(parts[2], "-")))
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

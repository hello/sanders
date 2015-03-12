package command

import (
	"fmt"
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/ec2"
	"github.com/crowdmob/goamz/elb"
	"github.com/mitchellh/cli"
	"strings"
	"sync"
)

type StatusCommand struct {
	Ui cli.ColoredUi
}

func (c *StatusCommand) Help() string {
	helpText := `Usage: hello status $appname ...`
	return strings.TrimSpace(helpText)
}

func (c *StatusCommand) Run(args []string) int {
	auth, _ := aws.EnvAuth()
	ecSquare := ec2.New(auth, aws.USEast)
	lb := elb.New(auth, aws.USEast)

	for _, elbName := range []string{"suripu-app-prod", "suripu-service-prod"} {
		c.Ui.Info(fmt.Sprintf("ELB: %s", elbName))
		lbResp, err := lb.DescribeInstanceHealth(elbName)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("%v", err))
			return 0
		}

		ids := make([]string, 0)
		amiIds := make([]string, 0)

		store := make(map[string]ec2.Instance)
		mu := sync.Mutex{}
		statusStore := make(map[string]string)
		nameStore := make(map[string]string)
		amiStore := make(map[string]ec2.Image)

		for _, state := range lbResp.InstanceStates {
			ids = append(ids, state.InstanceId)
			mu.Lock()
			statusStore[state.InstanceId] = state.State
			mu.Unlock()
		}

		instancesResp, err := ecSquare.DescribeInstances(ids, nil)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error: %s", err))
			return 1
		}

		for _, reservation := range instancesResp.Reservations {
			for _, instance := range reservation.Instances {
				store[instance.InstanceId] = instance
				amiIds = append(amiIds, instance.ImageId)
				for _, tag := range instance.Tags {
					if tag.Key == "Name" {
						nameStore[instance.InstanceId] = tag.Value
					} else {
						nameStore[instance.InstanceId] = "???"
					}
				}
			}
		}

		amiResp, err := ecSquare.Images(amiIds, nil)
		for _, image := range amiResp.Images {
			amiStore[image.Id] = image
		}

		for k, _ := range store {
			msg := fmt.Sprintf("→\t%s\t%s\t%s\t%s\t%s\t%s [%s]",
				statusStore[k],
				nameStore[k],
				store[k].PrivateDNSName,
				store[k].DNSName,
				store[k].InstanceId,
				amiStore[store[k].ImageId].Name,
				amiStore[store[k].ImageId].Description,
			)
			if statusStore[k] == "InService" {
				c.Ui.Info(msg)
			} else {
				c.Ui.Error(msg)
			}
		}
		c.Ui.Info("")
	}

	c.Ui.Output("")
	return 0
}

func (c *StatusCommand) Synopsis() string {
	return "Tell hello to get status for app"
}

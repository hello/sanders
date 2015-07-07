package command

import (
	"fmt"
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/autoscaling"
	"github.com/mitchellh/cli"
	"strings"
	"time"
)

const plan = `

Plan:
+++ ASG: %s
+++ LC: %s
+++ # of servers to deploy: %d

`

type CanaryCommand struct {
	Ui cli.ColoredUi
}

func (c *CanaryCommand) Help() string {
	helpText := `Usage: hello deploy`
	return strings.TrimSpace(helpText)
}

func (c *CanaryCommand) Run(args []string) int {

	config := &aws.Config{
		Region: "us-east-1",
	}
	service := autoscaling.New(config)

	version, err := c.Ui.Ask("Which version do you want to deploy to canary (ex 8.8.8): ")
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error reading version #: %s", err))
		return 1
	}

	c.Ui.Info(fmt.Sprintf("--> : %s", version))

	possibleLC := fmt.Sprintf("suripu-app-prod-%s", version)

	max := int64(3)
	describeLCReq := &autoscaling.DescribeLaunchConfigurationsInput{
		LaunchConfigurationNames: []*string{&possibleLC},
		MaxRecords:               &max,
	}

	lcsResp, err := service.DescribeLaunchConfigurations(describeLCReq)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	if len(lcsResp.LaunchConfigurations) == 0 {
		c.Ui.Error(fmt.Sprintf("No launch configuration found for version: %s", version))
		return 1
	}

	lcName := *lcsResp.LaunchConfigurations[0].LaunchConfigurationName
	c.Ui.Info(fmt.Sprintf("--> proceeding with LC : %s", lcName))

	groupnames := []*string{aws.String("suripu-app-rpod-canary")}

	describeASGreq := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: groupnames,
	}

	describeASGResp, err := service.DescribeAutoScalingGroups(describeASGreq)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	asg := describeASGResp.AutoScalingGroups[0]
	asgName := *asg.AutoScalingGroupName

	// Set to 0 first to kill the instance
	if len(asg.Instances) == 1 {
		_, err = c.update(service, 0, asgName, lcName)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Could not update LC: %s", err))
			return 1
		}

		ready := make(chan bool, 0)
		go c.check(service, asgName, ready)

		done := <-ready
		if !done {
			c.Ui.Error("Instance wasn't ready")
			return 1
		}
	}

	_, err = c.update(service, 1, asgName, lcName)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Could not update LC: %s", err))
		return 1
	}

	c.Ui.Info("Update autoscaling group request acknowledged")

	c.Ui.Info("Run: `sanders status` to monitor servers being attached to ELB")
	return 0
}

func (c *CanaryCommand) update(service *autoscaling.AutoScaling, desiredCapacity int64, asgName, lcName string) (*autoscaling.UpdateAutoScalingGroupOutput, error) {
	updateReq := &autoscaling.UpdateAutoScalingGroupInput{
		DesiredCapacity:         &desiredCapacity,
		AutoScalingGroupName:    aws.String(asgName),
		LaunchConfigurationName: &lcName,
		MinSize:                 &desiredCapacity,
		MaxSize:                 &desiredCapacity,
	}

	planMsg := fmt.Sprintf(plan, asgName, lcName, desiredCapacity)
	planIntro := "Executing plan:"
	if desiredCapacity > 0 {
		c.Ui.Info(planIntro)
		c.Ui.Info(planMsg)
	}

	return service.UpdateAutoScalingGroup(updateReq)
}

func (c *CanaryCommand) check(service *autoscaling.AutoScaling, asgName string, ready chan bool) {
	req := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{aws.String(asgName)},
	}
	for {
		resp, err := service.DescribeAutoScalingGroups(req)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("%s", err))
			ready <- false
			break
		}

		asg := resp.AutoScalingGroups[0]
		if len(asg.Instances) == 0 {
			ready <- true
		}
		c.Ui.Info("...")
		time.Sleep(2 * time.Second)
	}
}
func (c *CanaryCommand) Synopsis() string {
	return "deploy a new version of the app to the empty autoscaling group"
}

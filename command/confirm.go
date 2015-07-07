package command

import (
	"fmt"
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/autoscaling"
	"github.com/mitchellh/cli"
	"strconv"
	"strings"
)

type ConfirmCommand struct {
	Ui cli.ColoredUi
}

func (c *ConfirmCommand) Help() string {
	helpText := `Usage: hello up`
	return strings.TrimSpace(helpText)
}

func (c *ConfirmCommand) Run(args []string) int {
	plan := `

Plan:
+++ ASG: %s
+++ LC: %s
+++ # of servers to deploy: %d

`
	config := &aws.Config{
		Region: "us-east-1",
	}
	service := autoscaling.New(config)

	desiredCapacityByLCName := make(map[string]int64)

	desiredCapacityByLCName["suripu-app"] = int64(2)
	desiredCapacityByLCName["suripu-service"] = int64(4)

	version, err := c.Ui.Ask("Which version do you want to confirm (ex 8.8.8): ")
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error reading version #: %s", err))
		return 1
	}

	c.Ui.Info(fmt.Sprintf("--> : %s", version))

	possibleLCs := make([]*string, 2)
	apps := []string{"suripu-app", "suripu-service"}

	for idx, appName := range apps {
		str := fmt.Sprintf("%s-prod-%s", appName, version)
		possibleLCs[idx] = &str
	}

	max := int64(3)
	describeLCReq := &autoscaling.DescribeLaunchConfigurationsInput{
		LaunchConfigurationNames: possibleLCs,
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

	c.Ui.Output("")
	c.Ui.Output(fmt.Sprintf("Found the following matching Launch Configurations for version: %s:\n", version))
	for idx, stuff := range lcsResp.LaunchConfigurations {
		c.Ui.Info(fmt.Sprintf("[%d] %s", idx, *stuff.LaunchConfigurationName))
	}

	c.Ui.Output("")
	app, err := c.Ui.Ask("Launch configuration (LC) #: ")
	appIdx, _ := strconv.Atoi(app)

	if err != nil || appIdx >= len(lcsResp.LaunchConfigurations) {
		c.Ui.Error(fmt.Sprintf("Error reading app #: %s", err))
		return 1
	}

	lcName := *lcsResp.LaunchConfigurations[appIdx].LaunchConfigurationName
	c.Ui.Info(fmt.Sprintf("--> proceeding with LC : %s", lcName))

	parts := strings.Split(lcName, "-prod-")

	groupnames := make([]*string, 2)
	one := fmt.Sprintf("%s-prod", parts[0])
	two := fmt.Sprintf("%s-prod-green", parts[0])
	groupnames[0] = &one
	groupnames[1] = &two

	describeASGreq := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: groupnames,
	}

	describeASGResp, err := service.DescribeAutoScalingGroups(describeASGreq)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	desiredCapacity, found := desiredCapacityByLCName[parts[0]]
	if !found {
		c.Ui.Error(fmt.Sprintf("%s not found. Aborting", parts[0]))
		return 1
	}

	for _, asg := range describeASGResp.AutoScalingGroups {
		asgName := *asg.AutoScalingGroupName
		if *asg.LaunchConfigurationName == lcName {

			c.Ui.Warn(fmt.Sprintf("--- # of servers to deploy: %d", *asg.DesiredCapacity))
			c.Ui.Info(fmt.Sprintf("+++ # of servers to deploy: %d", desiredCapacity))

			if err != nil {
				c.Ui.Error(fmt.Sprintf("%s", err))
				return 1
			}

			ok, err := c.Ui.Ask("'ok' if you agree, anything else to cancel: ")
			if err != nil {
				c.Ui.Error(fmt.Sprintf("%s", err))
				return 1
			}

			if ok != "ok" {
				c.Ui.Warn("Cancelled.")
				return 0
			}

			maxSize := desiredCapacity * 2
			updateReq := &autoscaling.UpdateAutoScalingGroupInput{
				DesiredCapacity:         &desiredCapacity,
				AutoScalingGroupName:    &asgName,
				LaunchConfigurationName: &lcName,
				MinSize:                 &desiredCapacity,
				MaxSize:                 &maxSize,
			}

			c.Ui.Info("Executing plan:")
			c.Ui.Info(fmt.Sprintf(plan, asgName, lcName, *updateReq.DesiredCapacity))
			_, err = service.UpdateAutoScalingGroup(updateReq)
			if err != nil {
				c.Ui.Error(fmt.Sprintf("%s", err))
				return 1
			}

			// fmt.Println(*updateReq.AutoScalingGroupName)

			c.Ui.Info("Update autoscaling group request acknowledged")
		}
	}

	c.Ui.Info("Run: `sanders status` to monitor servers being attached to ELB")
	return 0
}

func (c *ConfirmCommand) Synopsis() string {
	return "confirms the given version is good and increase number of instances"
}

package command

import (
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/autoscaling"
	"github.com/mitchellh/cli"
	// "github.com/mitchellh/packer/packer"
	"fmt"
	// "sort"
	"strconv"
	"strings"
)

type DeployCommand struct {
	Ui cli.ColoredUi
}

func (c *DeployCommand) Help() string {
	helpText := `Usage: hello deploy`
	return strings.TrimSpace(helpText)
}

func (c *DeployCommand) Run(args []string) int {

	plan := `

Plan:
+++ ASG: %s
+++ LC: %s
+++ # of servers to deploy: %d

`
	creds, _ := aws.EnvCreds()
	service := autoscaling.New(creds, "us-east-1", nil)

	version, err := c.Ui.Ask("Which version do you want to deploy (ex 8.8.8): ")
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error reading version #: %s", err))
		return 1
	}

	c.Ui.Info(fmt.Sprintf("--> : %s", version))

	desiredCapacityByLCName := make(map[string]int)
	desiredCapacityByLCName["suripu-app"] = 2
	desiredCapacityByLCName["suripu-service"] = 2
	desiredCapacityByLCName["suripu-workers"] = 1

	possibleLCs := make([]string, 3)
	apps := []string{"suripu-app", "suripu-service", "suripu-workers"}

	for idx, appName := range apps {
		possibleLCs[idx] = fmt.Sprintf("%s-prod-%s", appName, version)
	}

	describeLCReq := &autoscaling.LaunchConfigurationNamesType{
		LaunchConfigurationNames: possibleLCs,
		MaxRecords:               aws.Integer(3),
	}

	lcsResp, err := service.DescribeLaunchConfigurations(describeLCReq)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
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

	groupnames := make([]string, 2)
	groupnames[0] = fmt.Sprintf("%s-prod", parts[0])
	groupnames[1] = fmt.Sprintf("%s-prod-green", parts[0])

	describeASGreq := &autoscaling.AutoScalingGroupNamesType{
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
		if *asg.DesiredCapacity == 0 {
			// c.Ui.Info(fmt.Sprintf("Update ASG %s with launch configuration:", asgName))

			c.Ui.Warn(fmt.Sprintf(plan, asgName, lcName, desiredCapacity))

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
			updateReq := &autoscaling.UpdateAutoScalingGroupType{
				DesiredCapacity:         aws.Integer(desiredCapacity),
				AutoScalingGroupName:    aws.String(asgName),
				LaunchConfigurationName: aws.String(lcName),
				MinSize:                 aws.Integer(desiredCapacity),
				MaxSize:                 aws.Integer(desiredCapacity * 2),
			}

			c.Ui.Info("Executing plan:")
			c.Ui.Info(fmt.Sprintf(plan, asgName, lcName, *updateReq.DesiredCapacity))
			err = service.UpdateAutoScalingGroup(updateReq)
			if err != nil {
				c.Ui.Error(fmt.Sprintf("%s", err))
				return 1
			}
			c.Ui.Info("Update autoscaling group request acknowledged")

			continue
		}
		c.Ui.Warn(fmt.Sprintf("%s ignored because desired capacity is > 0", asgName))
	}

	c.Ui.Info("Run: `sanders status` to monitor servers being attached to ELB")
	return 0
}

func (c *DeployCommand) Synopsis() string {
	return "deploy a new version of the app to the empty autoscaling group"
}

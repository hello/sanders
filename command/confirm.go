package command

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hello/sanders/core"
	"github.com/mitchellh/cli"
	"strconv"
	"strings"
)

type ConfirmCommand struct {
	Ui       cli.ColoredUi
	Notifier BasicNotifier
	Apps     []core.SuripuApp
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
		Region: aws.String("us-east-1"),
	}
	service := autoscaling.New(session.New(), config)

	version, err := c.Ui.Ask("Which version do you want to confirm (ex 8.8.8): ")
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error reading version #: %s", err))
		return 1
	}

	c.Ui.Info(fmt.Sprintf("--> : %s", version))

	possibleLCs := make([]*string, len(c.Apps))

	var appNameMap map[string]core.SuripuApp
	appNameMap = make(map[string]core.SuripuApp)

	for idx, app := range c.Apps {
		str := fmt.Sprintf("%s-prod-%s", app.Name, version)
		possibleLCs[idx] = &str
		appNameMap[app.Name] = app
	}

	max := int64(len(c.Apps))
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

	selectedApp := appNameMap[parts[0]]

	groupnames := make([]*string, 2)
	one := fmt.Sprintf("%s-prod", selectedApp.Name)
	two := fmt.Sprintf("%s-prod-green", selectedApp.Name)
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

	desiredCapacity := selectedApp.TargetDesiredCapacity

	for _, asg := range describeASGResp.AutoScalingGroups {
		asgName := *asg.AutoScalingGroupName
		if *asg.LaunchConfigurationName == lcName && *asg.DesiredCapacity != desiredCapacity {

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
			deployAction := NewDeployAction("confirm", asgName, lcName, desiredCapacity)
			c.Notifier.Notify(deployAction)
			// fmt.Println(*updateReq.AutoScalingGroupName)

			c.Ui.Info("Update autoscaling group request acknowledged")
			return 0
		}
	}

	c.Ui.Info("Run: `sanders status` to monitor servers being attached to ELB")
	return 0
}

func (c *ConfirmCommand) Synopsis() string {
	return "confirms the given version is good and increase number of instances"
}

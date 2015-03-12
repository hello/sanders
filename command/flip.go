package command

import (
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/autoscaling"
	"github.com/mitchellh/cli"
	// "github.com/mitchellh/packer/packer"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type FlipCommand struct {
	Ui cli.ColoredUi
}

func (c *FlipCommand) Help() string {
	helpText := `Usage: hello flip $appname ...`
	return strings.TrimSpace(helpText)
}

func (c *FlipCommand) Run(args []string) int {

	apps := []string{"suripu-app", "suripu-service", "suripu-workers"}
	for idx, appName := range apps {
		c.Ui.Info(fmt.Sprintf("[%d] %s", idx, appName))
	}

	app, err := c.Ui.Ask("App #: ")
	appIdx, _ := strconv.Atoi(app)

	if err != nil || appIdx >= len(apps) {
		c.Ui.Error(fmt.Sprintf("Error reading app #: %s", err))
		return 1
	}

	creds, _ := aws.EnvCreds()
	cli := autoscaling.New(creds, "us-east-1", nil)

	groupnames := make([]string, 2)
	groupnames[0] = fmt.Sprintf("%s-prod", apps[appIdx])
	groupnames[1] = fmt.Sprintf("%s-prod-green", apps[appIdx])

	req := &autoscaling.AutoScalingGroupNamesType{
		AutoScalingGroupNames: groupnames,
	}

	resp, err := cli.DescribeAutoScalingGroups(req)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	for _, asg := range resp.AutoScalingGroups {
		c.Ui.Info(fmt.Sprintf("ASG: %s", *asg.AutoScalingGroupName))
		c.Ui.Info(fmt.Sprintf("\tDesired capacity: %d", asg.DesiredCapacity))
		c.Ui.Info(fmt.Sprintf("\tInstances: %d", len(asg.Instances)))
		c.Ui.Info(fmt.Sprintf("\tVPC id: %v", *asg.VPCZoneIdentifier))
		c.Ui.Info("")
	}

	lcsResp, err := cli.DescribeLaunchConfigurations(nil)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	matchingLcs := make([]string, 0)

	for _, lc := range lcsResp.LaunchConfigurations {
		lcName := *lc.LaunchConfigurationName
		if strings.HasPrefix(lcName, groupnames[0]) {
			matchingLcs = append(matchingLcs, lcName)
		}
	}

	var strSlice sort.StringSlice = matchingLcs
	sort.Sort(sort.Reverse(strSlice[:]))

	for i, lc := range strSlice {
		if i < 5 {
			c.Ui.Info(fmt.Sprintf("LC: %s", lc))
		}
	}
	c.Ui.Info("")

	asgNameToUpdate, err := c.Ui.Ask("ASG to update:")
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	for _, asg := range resp.AutoScalingGroups {

		asgName := *asg.AutoScalingGroupName
		if asgName == asgNameToUpdate {
			msg := fmt.Sprintf("Update ASG %s with launch configuration:", asgName)
			lcName, err := c.Ui.Ask(msg)
			if err != nil {
				c.Ui.Error(fmt.Sprintf("%s", err))
				return 1
			}

			desiredCapacityStr, err := c.Ui.Ask("DesiredCapacity:")
			if err != nil {
				c.Ui.Error(fmt.Sprintf("%s", err))
				return 1
			}

			desiredCapacity, err := strconv.Atoi(desiredCapacityStr)
			if err != nil {
				c.Ui.Error(fmt.Sprintf("%s", err))
				return 1
			}

			req := &autoscaling.UpdateAutoScalingGroupType{
				DesiredCapacity:         aws.Integer(desiredCapacity),
				AutoScalingGroupName:    aws.String(asgName),
				LaunchConfigurationName: aws.String(lcName),
				MinSize:                 aws.Integer(desiredCapacity),
				MaxSize:                 aws.Integer(2),
			}

			err = cli.UpdateAutoScalingGroup(req)

			if err != nil {
				c.Ui.Error(fmt.Sprintf("%s", err))
				return 1
			}
		}
	}

	return 0
}

func (c *FlipCommand) Synopsis() string {
	return "Tell hello to deploy a new version of the app"
}

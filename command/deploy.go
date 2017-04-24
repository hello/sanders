package command

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hello/sanders/core"
	"github.com/mitchellh/cli"
	"strings"
)

type DeployCommand struct {
	Ui       cli.ColoredUi
	Notifier BasicNotifier
	Apps     []core.SuripuApp
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
	config := &aws.Config{
		Region: aws.String("us-east-1"),
	}
	service := autoscaling.New(session.New(), config)

	desiredCapacity := int64(1)

	appSelector := core.NewCliAppSelector(c.Ui)
	selectedApp, err := appSelector.Choose(c.Apps)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	lcSelector := core.NewCliLaunchConfigurationSelector(c.Ui, service)

	lcName, err := lcSelector.Choose(selectedApp)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Info(fmt.Sprintf("--> proceeding with LC : %s", lcName))

	appName := selectedApp.Name

	groupnames := make([]*string, 2)
	one := fmt.Sprintf("%s-prod", appName)
	two := fmt.Sprintf("%s-prod-green", appName)
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

	for _, asg := range describeASGResp.AutoScalingGroups {
		asgName := *asg.AutoScalingGroupName
		if *asg.DesiredCapacity == 0 {
			c.Ui.Info(fmt.Sprintf("Update ASG %s with launch configuration:", asgName))

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

			maxSize := desiredCapacity * 2
			updateReq := &autoscaling.UpdateAutoScalingGroupInput{
				DesiredCapacity:         &desiredCapacity,
				AutoScalingGroupName:    &asgName,
				LaunchConfigurationName: &lcName,
				MinSize:                 &desiredCapacity,
				MaxSize:                 &maxSize,
			}

			deployAction := NewDeployAction("deploy", asgName, lcName, *updateReq.DesiredCapacity)
			c.Ui.Info("Executing plan:")
			c.Ui.Info(fmt.Sprintf(plan, asgName, lcName, *updateReq.DesiredCapacity))

			_, err = service.UpdateAutoScalingGroup(updateReq)
			if err != nil {
				c.Ui.Error(fmt.Sprintf("%s", err))
				return 1
			}

			c.Notifier.Notify(deployAction)

			tags := []core.Tag{
				{
					AsgName:   asgName,
					TagName:   "Launch Configuration",
					TagValue:  lcName,
					Propagate: true,
				},
				{
					AsgName:   asgName,
					TagName:   "Name",
					TagValue:  fmt.Sprintf("%s-prod", appName),
					Propagate: true,
				},
				{
					AsgName:   asgName,
					TagName:   "Env",
					TagValue:  "prod",
					Propagate: true,
				},
				{
					AsgName:   asgName,
					TagName:   "Service",
					TagValue:  appName,
					Propagate: true,
				},
			}

			respTag, err := c.updateASGTags(service, tags)
			if err != nil {
				c.Ui.Error(fmt.Sprintf("%s", err))
				return 1
			}

			if respTag != nil {
				c.Ui.Info("Tags successfully updated.")
			}

			c.Ui.Info(fmt.Sprintf("Update autoscaling group %s request acknowledged", asgName))
			return 0
		}
		c.Ui.Warn(fmt.Sprintf("%s ignored because desired capacity is > 0", asgName))
	}

	c.Ui.Info("Run: `sanders status` to monitor servers being attached to ELB")
	return 0
}

func (c *DeployCommand) updateASGTags(service *autoscaling.AutoScaling, tagsToUpdate []core.Tag) (*autoscaling.CreateOrUpdateTagsOutput, error) {

	tags := make([]*autoscaling.Tag, 0)

	for _, tag := range tagsToUpdate {
		awsTag := &autoscaling.Tag{ // Required
			Key:               aws.String(tag.TagName), // Required
			PropagateAtLaunch: aws.Bool(tag.Propagate),
			ResourceId:        aws.String(tag.AsgName),
			ResourceType:      aws.String("auto-scaling-group"),
			Value:             aws.String(tag.TagValue),
		}

		tags = append(tags, awsTag)
	}

	//Tag the ASG so version number can be passed to instance
	params := &autoscaling.CreateOrUpdateTagsInput{
		Tags: tags,
	}
	resp, err := service.CreateOrUpdateTags(params)

	return resp, err
}

func (c *DeployCommand) Synopsis() string {
	return "deploy a new version of the app to the empty autoscaling group"
}

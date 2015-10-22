package command

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/mitchellh/cli"
	"sort"
	"strconv"
	"strings"
)

type ByLCTime []*autoscaling.LaunchConfiguration

func (s ByLCTime) Len() int {
	return len(s)
}
func (s ByLCTime) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByLCTime) Less(i, j int) bool {
	return s[i].CreatedTime.Unix() < s[j].CreatedTime.Unix()
}

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
	config := &aws.Config{
		Region: aws.String("us-east-1"),
	}
	service := autoscaling.New(config)

	desiredCapacity := int64(1)

	c.Ui.Output("Which app would you like to deploy?")

	suripuApps := []suripuApp{
		suripuApp{name: "suripu-app"},
		suripuApp{name: "suripu-service"},
		suripuApp{name: "suripu-workers"},
		suripuApp{name: "suripu-admin"},
	}

	for idx, app := range suripuApps {
		c.Ui.Output(fmt.Sprintf("[%d] %s", idx, app.name))
	}

	appSel, err := c.Ui.Ask("Select an app #: ")
	appIdx, _ := strconv.Atoi(appSel)

	if err != nil || appIdx >= len(suripuApps) {
		c.Ui.Error(fmt.Sprintf("Incorrect app selection: %s\n", err))
		return 1
	}

	lcParams := &autoscaling.DescribeLaunchConfigurationsInput{
		MaxRecords: aws.Int64(100),
	}

	pageNum := 0
	appPossibleLCs := make([]*autoscaling.LaunchConfiguration, 0)

	pageErr := service.DescribeLaunchConfigurationsPages(lcParams, func(page *autoscaling.DescribeLaunchConfigurationsOutput, lastPage bool) bool {
		pageNum++
		if len(page.LaunchConfigurations) == 0 {
			c.Ui.Error(fmt.Sprintf("No launch configuration found for app: %s", suripuApps[appIdx].name))
			return false
		}

		for _, stuff := range page.LaunchConfigurations {
			if strings.HasPrefix(*stuff.LaunchConfigurationName, suripuApps[appIdx].name) {
				appPossibleLCs = append(appPossibleLCs, stuff)
			}
		}
		return pageNum <= 2 //Allow for 200 possible LCs
	})

	if pageErr != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	c.Ui.Info("Latest 5 Launch Configurations")
	c.Ui.Info(" #\tName:                               \tCreated At:")
	c.Ui.Info("---|---------------------------------------|---------------")
	sort.Sort(sort.Reverse(ByLCTime(appPossibleLCs)))

	numLCs := Min(len(appPossibleLCs), 5)
	for lcIdx := 0; lcIdx < numLCs; lcIdx++ {
		c.Ui.Info(fmt.Sprintf("[%d]\t%-36s\t%s", lcIdx, *appPossibleLCs[lcIdx].LaunchConfigurationName, appPossibleLCs[lcIdx].CreatedTime.String()))
	}

	c.Ui.Output("")
	lc, err := c.Ui.Ask("Select a launch configuration (LC) #: ")
	lcNum, _ := strconv.Atoi(lc)

	if err != nil || lcNum >= len(appPossibleLCs) {
		c.Ui.Error(fmt.Sprintf("Error reading app #: %s", err))
		return 1
	}

	lcName := *appPossibleLCs[lcNum].LaunchConfigurationName
	c.Ui.Info(fmt.Sprintf("--> proceeding with LC : %s", lcName))

	appName := suripuApps[appIdx].name

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

			c.Ui.Info("Executing plan:")
			c.Ui.Info(fmt.Sprintf(plan, asgName, lcName, *updateReq.DesiredCapacity))
			_, err = service.UpdateAutoScalingGroup(updateReq)
			if err != nil {
				c.Ui.Error(fmt.Sprintf("%s", err))
				return 1
			}

			//Tag the ASG so version number can be passed to instance
			params := &autoscaling.CreateOrUpdateTagsInput{
				Tags: []*autoscaling.Tag{ // Required
					{ // Required
						Key:               aws.String("LC_Name"), // Required
						PropagateAtLaunch: aws.Bool(true),
						ResourceId:        &asgName,
						ResourceType:      aws.String("auto-scaling-group"),
						Value:             &lcName,
					},
				},
			}
			resp, err := service.CreateOrUpdateTags(params)

			if err != nil {
				c.Ui.Error(fmt.Sprintf("%s", err))
				return 1
			}

			if resp != nil {
				c.Ui.Info("Added 'LC_Name' tag to ASG.")
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

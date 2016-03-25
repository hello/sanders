package command

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hello/sanders/ui"
	"strings"
	"time"
	"strconv"
	"sort"
)

const plan = `

Plan:
+++ ASG: %s
+++ LC: %s
+++ # of servers to deploy: %d

`

type CanaryCommand struct {
	Ui       ui.ProgressUi
	Notifier BasicNotifier
}

func (c *CanaryCommand) Help() string {
	helpText := `Usage: hello deploy`
	return strings.TrimSpace(helpText)
}

func (c *CanaryCommand) Run(args []string) int {

	config := &aws.Config{
		Region: aws.String("us-east-1"),
	}
	service := autoscaling.New(session.New(), config)

	//desiredCapacity := int64(1)

	c.Ui.Output("Which app would you like to deploy?")

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

	groupnames := []*string{aws.String(fmt.Sprintf("%s-prod-canary", appName))}

	describeASGreq := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: groupnames,
	}

	describeASGResp, err := service.DescribeAutoScalingGroups(describeASGreq)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	if len(describeASGResp.AutoScalingGroups) < 1 {
		c.Ui.Error(fmt.Sprintf("There doesn't appear to be an ASG for %s. Exiting...", appName))
		return 1
	}

	asg := describeASGResp.AutoScalingGroups[0]
	asgName := *asg.AutoScalingGroupName

	c.Ui.Info(fmt.Sprintf("LC: %s\nASG: %s", lcName, asgName))

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

	deployAction := NewDeployAction("canary", asgName, lcName, 1)
	_, err = c.update(service, 1, asgName, lcName)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Could not update LC: %s", err))
		return 1
	}

	c.Notifier.Notify(deployAction)
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

	resp, err := service.UpdateAutoScalingGroup(updateReq)

	if (desiredCapacity > 0) && (err == nil) {
		respTag, err := c.updateASGTag(service, asgName, "Launch Configuration", lcName, true)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("%s", err))
			return resp, err
		}

		if respTag != nil {
			c.Ui.Info("Added 'Launch Configuration' tag to ASG.")
		}

		respTag, err = c.updateASGTag(service, asgName, "Name", "suripu-app-canary", true)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("%s", err))
			return resp, err
		}

		if respTag != nil {
			c.Ui.Info("Added 'Name' tag to ASG.")
		}
	}

	return resp, err
}

func (c *CanaryCommand) updateASGTag(service *autoscaling.AutoScaling, asgName string, tagName string, tagValue string, propagate bool) (*autoscaling.CreateOrUpdateTagsOutput, error){

	//Tag the ASG so version number can be passed to instance
	params := &autoscaling.CreateOrUpdateTagsInput{
		Tags: []*autoscaling.Tag{// Required
			{// Required
				Key:               aws.String(tagName), // Required
				PropagateAtLaunch: aws.Bool(propagate),
				ResourceId:        aws.String(asgName),
				ResourceType:      aws.String("auto-scaling-group"),
				Value:             aws.String(tagValue),
			},
		},
	}
	resp, err := service.CreateOrUpdateTags(params)

	return resp, err
}

func (c *CanaryCommand) check(service *autoscaling.AutoScaling, asgName string, ready chan bool) {
	req := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{aws.String(asgName)},
	}

	symbols := []int{
		0x1f550,
		0x1f551,
		0x1f552,
		0x1f553,
		0x1f554,
		0x1f555,
		0x1f556,
		0x1f557,
		0x1f558,
		0x1f559,
		0x1f55a,
		0x1f55b,
	}
	i := 0
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
		offset := i % len(symbols)
		c.Ui.Progress(fmt.Sprintf(" %c", symbols[offset]))
		i += 1
		time.Sleep(1 * time.Second)
		offset = i % len(symbols)
		c.Ui.Progress(fmt.Sprintf(" %c", symbols[offset]))
		i += 1
		time.Sleep(1 * time.Second)
	}
}

func (c *CanaryCommand) Synopsis() string {
	return "deploy a new version of the app to the empty autoscaling group"
}

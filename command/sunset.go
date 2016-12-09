package command

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/mitchellh/cli"
	// "github.com/mitchellh/packer/packer"
	"fmt"
	// "sort"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/hello/sanders/core"
	"strconv"
	"strings"
)

type SunsetCommand struct {
	Ui       cli.ColoredUi
	Notifier BasicNotifier
	Apps     []core.SuripuApp
}

func (c *SunsetCommand) Help() string {
	helpText := `Usage: hello sunset`
	return strings.TrimSpace(helpText)
}

func (c *SunsetCommand) Run(args []string) int {

	plan := `

Plan:
--- ASG: %s
--- LC: %s
--- # of servers: %d

`

	config := &aws.Config{
		Region: aws.String("us-east-1"),
	}

	service := autoscaling.New(session.New(), config)

	c.Ui.Output("Which of the following apps do you want to sunset?\n")

	for idx, app := range c.Apps {
		c.Ui.Output(fmt.Sprintf("[%d] %s", idx, app.Name))
	}
	appSel, err := c.Ui.Ask("Select an app #: ")
	appIdx, _ := strconv.Atoi(appSel)

	if err != nil || appIdx >= len(c.Apps) {
		c.Ui.Error(fmt.Sprintf("Incorrect app selection: %s\n", err))
		return 1
	}

	c.Ui.Info(fmt.Sprintf("--> proceeding to sunset app: %s\n", c.Apps[appIdx].Name))

	groupnames := make([]*string, 2)
	one := fmt.Sprintf("%s-prod", c.Apps[appIdx].Name)
	two := fmt.Sprintf("%s-prod-green", c.Apps[appIdx].Name)
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

	instancesPerASG := make(map[string]*autoscaling.Group)

	asgs := make([]string, 0)
	for _, asg := range describeASGResp.AutoScalingGroups {
		asgName := *asg.AutoScalingGroupName
		asgs = append(asgs, fmt.Sprintf("%s", asgName))
		instancesPerASG[asgName] = asg
	}

	allASGsAtDesiredCapacity := true
	c.Ui.Output(fmt.Sprintf("ASG matching app : %s\n", c.Apps[appIdx].Name))
	for idx, asgName := range asgs {
		asg, _ := instancesPerASG[asgName]
		parts := strings.Split(*asg.LaunchConfigurationName, "-prod-")
		c.Ui.Info(fmt.Sprintf("[%d] %s (%d instances running %s)", idx, asgName, len(asg.Instances), parts[1]))
		if len(asg.Instances) < int(c.Apps[appIdx].TargetDesiredCapacity) {
			allASGsAtDesiredCapacity = false
		}
	}

	if allASGsAtDesiredCapacity == false {
		c.Ui.Output("")
		c.Ui.Error(fmt.Sprintf("All ASGs are not at desired capacity (%d). Ensure you have confirmed your deploy.", c.Apps[appIdx].TargetDesiredCapacity))

		c.Ui.Warn("Would you like to override and sunset an ASG anyway?")
		ok, err := c.Ui.Ask("'ok' if you would like to override, anything else to cancel: ")
		if err != nil {
			c.Ui.Error(fmt.Sprintf("%s", err))
			return 1
		}

		if ok != "ok" {
			c.Ui.Warn("Cancelled.")
			return 0
		}
	}

	c.Ui.Output("")
	choiceStr, err := c.Ui.Ask("Choice: #")
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%v", err))
		return 1
	}

	choice, _ := strconv.Atoi(choiceStr)
	if choice >= len(asgs) {
		c.Ui.Error(fmt.Sprintf("Error reading app #: %s", err))
		return 1
	}

	sunsetAsg := asgs[choice]

	asg := instancesPerASG[sunsetAsg]

	if len(asg.Instances) == 0 {
		c.Ui.Warn(fmt.Sprintf("ASG %s already has 0 instances, bailing.", sunsetAsg))
		return 0
	}

	completePlan := fmt.Sprintf(plan, sunsetAsg, "N/A", 0)
	c.Ui.Warn(completePlan)

	ok, err := c.Ui.Ask("'ok' if you agree, anything else to cancel: ")
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	if ok != "ok" {
		c.Ui.Warn("Cancelled.")
		return 0
	}

	numServers := int64(0)
	updateReq := &autoscaling.UpdateAutoScalingGroupInput{
		DesiredCapacity:      &numServers,
		AutoScalingGroupName: &sunsetAsg,
		MinSize:              &numServers,
		MaxSize:              &numServers,
	}

	deployAction := NewDeployAction("sunset", sunsetAsg, "-", numServers)

	c.Ui.Info("Executing plan:")
	c.Ui.Info(fmt.Sprintf(plan, sunsetAsg, "N/A", *updateReq.DesiredCapacity))
	_, err = service.UpdateAutoScalingGroup(updateReq)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}
	c.Ui.Info("Update autoscaling group request acknowledged")

	c.Notifier.Notify(deployAction)
	c.Ui.Info("Run: `sanders status` to monitor servers being attached to ELB")
	return 0
}

func (c *SunsetCommand) Synopsis() string {
	return "sunset instances inside one autoscaling group"
}

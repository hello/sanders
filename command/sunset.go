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

type SunsetCommand struct {
	Ui cli.ColoredUi
}

func (c *SunsetCommand) Help() string {
	helpText := `Usage: hello sunset`
	return strings.TrimSpace(helpText)
}

func (c *SunsetCommand) Run(args []string) int {

	plan := `

--> Plan:
--> ASG: %s
--> LC: %s
--> # of servers: %d

`

	creds, _ := aws.EnvCreds()
	service := autoscaling.New(creds, "us-east-1", nil)

	apps := []string{"suripu-app", "suripu-service", "suripu-workers"}
	for idx, appName := range apps {
		c.Ui.Info(fmt.Sprintf("[%d] %s", idx, appName))
	}

	choiceStr, err := c.Ui.Ask("Choice: #")
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%v", err))
		return 1
	}

	choice, _ := strconv.Atoi(choiceStr)
	if choice >= len(apps) {
		c.Ui.Error(fmt.Sprintf("Error reading app #: %s", err))
		return 1
	}

	groupnames := make([]string, 2)
	groupnames[0] = fmt.Sprintf("%s-prod", apps[choice])
	groupnames[1] = fmt.Sprintf("%s-prod-green", apps[choice])

	describeASGreq := &autoscaling.AutoScalingGroupNamesType{
		AutoScalingGroupNames: groupnames,
	}

	describeASGResp, err := service.DescribeAutoScalingGroups(describeASGreq)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	instancesPerASG := make(map[string]autoscaling.AutoScalingGroup)

	asgs := make([]string, 0)
	for _, asg := range describeASGResp.AutoScalingGroups {
		asgName := *asg.AutoScalingGroupName
		asgs = append(asgs, fmt.Sprintf("%s", asgName))
		instancesPerASG[asgName] = asg
	}

	for idx, asgName := range asgs {
		asg, _ := instancesPerASG[asgName]
		c.Ui.Info(fmt.Sprintf("[%d] %s (%d instances)", idx, asgName, len(asg.Instances)))
	}

	choiceStr, err = c.Ui.Ask("Choice: #")
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%v", err))
		return 1
	}

	choice, _ = strconv.Atoi(choiceStr)
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

	c.Ui.Warn(fmt.Sprintf(plan, sunsetAsg, "N/A", 0))

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
		DesiredCapacity:      aws.Integer(0),
		AutoScalingGroupName: aws.String(sunsetAsg),
		MinSize:              aws.Integer(0),
		MaxSize:              aws.Integer(0),
	}

	c.Ui.Info("Executing plan:")
	c.Ui.Info(fmt.Sprintf(plan, sunsetAsg, "N/A", *updateReq.DesiredCapacity))
	err = service.UpdateAutoScalingGroup(updateReq)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}
	c.Ui.Info("Update autoscaling group request acknowledged")

	c.Ui.Info("Run: `sanders status` to monitor servers being attached to ELB")
	return 0
}

func (c *SunsetCommand) Synopsis() string {
	return "Tell hello to deploy a new version of the app"
}

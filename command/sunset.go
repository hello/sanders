package command

import (
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/autoscaling"
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

Plan:
--- ASG: %s
--- LC: %s
--- # of servers: %d

`

	creds, _ := aws.EnvCreds()
	config := &aws.Config{
		Credentials: creds,
		Region:      "us-east-1",
	}

	service := autoscaling.New(config)

	apps := []string{"suripu-app", "suripu-service", "suripu-workers"}
	c.Ui.Output("Which of the following apps do you want to sunset?\n")
	for idx, appName := range apps {
		c.Ui.Info(fmt.Sprintf("[%d] %s", idx, appName))
	}

	c.Ui.Output("")
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

	c.Ui.Info(fmt.Sprintf("--> proceeding to sunset app: %s\n", apps[choice]))

	groupnames := make([]*string, 2)
	one := fmt.Sprintf("%s-prod", apps[choice])
	two := fmt.Sprintf("%s-prod-green", apps[choice])
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

	instancesPerASG := make(map[string]*autoscaling.AutoScalingGroup)

	asgs := make([]string, 0)
	for _, asg := range describeASGResp.AutoScalingGroups {
		asgName := *asg.AutoScalingGroupName
		asgs = append(asgs, fmt.Sprintf("%s", asgName))
		instancesPerASG[asgName] = asg
	}

	c.Ui.Output(fmt.Sprintf("ASG matching app : %s\n", apps[choice]))
	for idx, asgName := range asgs {
		asg, _ := instancesPerASG[asgName]
		c.Ui.Info(fmt.Sprintf("[%d] %s (%d instances)", idx, asgName, len(asg.Instances)))
	}

	c.Ui.Output("")
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

	numServers := int64(0)
	updateReq := &autoscaling.UpdateAutoScalingGroupInput{
		DesiredCapacity:      &numServers,
		AutoScalingGroupName: &sunsetAsg,
		MinSize:              &numServers,
		MaxSize:              &numServers,
	}

	c.Ui.Info("Executing plan:")
	c.Ui.Info(fmt.Sprintf(plan, sunsetAsg, "N/A", *updateReq.DesiredCapacity))
	_, err = service.UpdateAutoScalingGroup(updateReq)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}
	c.Ui.Info("Update autoscaling group request acknowledged")

	c.Ui.Info("Run: `sanders status` to monitor servers being attached to ELB")
	return 0
}

func (c *SunsetCommand) Synopsis() string {
	return "sunset instances inside one autoscaling group"
}

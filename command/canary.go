package command

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hello/sanders/ui"
	"strings"
	"time"
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

	version, err := c.Ui.Ask("Which version do you want to deploy to canary (ex 8.8.8): ")
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error reading version #: %s", err))
		return 1
	}

	c.Ui.Info(fmt.Sprintf("--> : %s", version))

	possibleLC := fmt.Sprintf("suripu-app-prod-%s", version)

	max := int64(3)
	describeLCReq := &autoscaling.DescribeLaunchConfigurationsInput{
		LaunchConfigurationNames: []*string{&possibleLC},
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

	lcName := *lcsResp.LaunchConfigurations[0].LaunchConfigurationName
	c.Ui.Info(fmt.Sprintf("--> proceeding with LC : %s", lcName))

	groupnames := []*string{aws.String("suripu-app-rpod-canary")}

	describeASGreq := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: groupnames,
	}

	describeASGResp, err := service.DescribeAutoScalingGroups(describeASGreq)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	asg := describeASGResp.AutoScalingGroups[0]
	asgName := *asg.AutoScalingGroupName

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

	return service.UpdateAutoScalingGroup(updateReq)
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

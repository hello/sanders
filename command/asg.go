package command

import (
	"github.com/awslabs/aws-sdk-go/aws"
	gasg "github.com/awslabs/aws-sdk-go/gen/autoscaling"
	"github.com/mitchellh/cli"

	"fmt"
	"strings"
)

type ASGCommand struct {
	Ui cli.ColoredUi
}

func (c *ASGCommand) Help() string {
	helpText := `Usage: hello build $appname ...`
	return strings.TrimSpace(helpText)
}

func (c *ASGCommand) Run(args []string) int {
	creds, _ := aws.EnvCreds()
	cli := gasg.New(creds, "us-east-1", nil)

	req := &gasg.UpdateAutoScalingGroupType{
		DesiredCapacity:         aws.Integer(0),
		AutoScalingGroupName:    aws.String("suripu-whatever-test"),
		LaunchConfigurationName: aws.String("suripu-app-prod-bbb"),
		MinSize:                 aws.Integer(0),
	}

	err := cli.UpdateAutoScalingGroup(req)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	return 0
}

func (c *ASGCommand) Synopsis() string {
	return "Tell hello to deploy a new version of the app"
}

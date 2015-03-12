package command

import (
	"github.com/crowdmob/goamz/autoscaling"
	"github.com/crowdmob/goamz/aws"
	"github.com/mitchellh/cli"
	// "github.com/mitchellh/packer/packer"
	"fmt"
	"strconv"
	"strings"
)

type BuildCommand struct {
	Ui cli.ColoredUi
}

func (c *BuildCommand) Help() string {
	helpText := `Usage: hello build $appname ...`
	return strings.TrimSpace(helpText)
}

func (c *BuildCommand) Run(args []string) int {

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

	version, err := c.Ui.Ask("Version #: ")

	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error reading app version: %s", err))
		return 1
	}

	envAuth, err := aws.EnvAuth()
	as := autoscaling.New(envAuth, aws.USEast)
	lc := autoscaling.LaunchConfiguration{
		LaunchConfigurationName:  fmt.Sprintf("%s-prod-%s", apps[appIdx], version),
		AssociatePublicIpAddress: true,
		EbsOptimized:             false,
		ImageId:                  "ami-32eab15a",
		InstanceType:             "m1.small",
		SecurityGroups:           []string{"sg-11ac0e75"},
		KeyName:                  "vpc-prod",
		InstanceMonitoring:       "true",
	}

	resp, err := as.CreateLaunchConfiguration(lc)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	c.Ui.Info(fmt.Sprintf("RequestId: %s", resp.RequestId))
	c.Ui.Info(fmt.Sprintf("LC name: %s", lc.LaunchConfigurationName))
	return 0
}

func (c *BuildCommand) Synopsis() string {
	return "Tell hello to deploy a new version of the app"
}

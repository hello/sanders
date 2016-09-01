package command

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/hello/sanders/setup"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/multistep"
	"strings"
)

type SetupCommand struct {
	Ui     cli.ColoredUi
	Config *aws.Config
}

func (c *SetupCommand) Help() string {
	helpText := `Usage: setup`
	return strings.TrimSpace(helpText)
}

func (c *SetupCommand) err(err error) int {
	c.Ui.Error("You might have to manually clean up state changes")
	c.Ui.Error(fmt.Sprintf("%s", err))
	return 1
}

func (c *SetupCommand) Run(args []string) int {

	sess := session.New()
	asg := autoscaling.New(sess, c.Config)
	ec2srv := ec2.New(sess, c.Config)
	elbsrv := elb.New(sess, c.Config)

	state := new(multistep.BasicStateBag)
	state.Put("ui", c.Ui)
	state.Put("asg", asg)
	state.Put("ec2", ec2srv)
	state.Put("elb", elbsrv)

	appName, err := c.Ui.Ask("New application name? Ex: suripu-service, supichi, …\n")
	if err != nil {
		return c.err(err)
	}
	vpcId := "vpc-961464f3"
	appInPort := int64(8080)
	azs := []string{"us-east-1a", "us-east-1b"}
	appName = strings.TrimSpace(appName)
	subnets := []string{
		"subnet-28c6565f", // 1A
		"subnet-da02b383", // 1B
	}
	// Build the steps
	steps := []multistep.Step{
		&setup.StepCreateSecurityGroups{
			AppName:   appName,
			VpcId:     vpcId,
			AppInPort: appInPort,
		},
		&setup.StepCreateELB{
			AppName:    appName,
			ElbOutPort: appInPort,
			ElbInPort:  int64(443),
			Subnets:    subnets,
		},
		&setup.StepLaunchConfiguration{
			AppName:        appName,
			ImageId:        "ami-d06267ba",
			SecurityGroups: []string{},
			KeyName:        "vpc-root",
			InstanceType:   "c3.large",
		},
		&setup.StepCreateAutoScalingGroups{
			AppName: appName,
			Azs:     azs,
			Subnets: subnets,
		},
	}

	//
	runner := &multistep.BasicRunner{
		Steps: steps,
	}
	runner.Run(state)

	// If there was an error, return that
	if rawErr, ok := state.GetOk("error"); ok {
		c.Ui.Error(fmt.Sprintf("%s\n", rawErr))
		return 1
	}
	return 0
}

func (c *SetupCommand) Synopsis() string {
	return "Creates a launch configuration based on selected parameters."
}

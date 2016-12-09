package command

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hello/sanders/core"
	"github.com/mitchellh/cli"
	"strings"
	"time"
)

type CreateCommand struct {
	Ui          cli.ColoredUi
	Notifier    BasicNotifier
	AmiSelector core.AmiSelector
	Ec2Service  *ec2.EC2
	S3Service   *s3.S3
	AsgService  *autoscaling.AutoScaling
	KeyService  core.KeyService
	Apps        []core.SuripuApp
}

func (c *CreateCommand) Help() string {
	helpText := `Usage: create [--emergency] [--canary]
	--emergency		Create specially named Launch Config for emergency situations ONLY.
	--canary		Create a Launch Config for a canary build. (Not necessary for canary deploys)`
	return strings.TrimSpace(helpText)
}

func (c *CreateCommand) Run(args []string) int {

	var isEmergency bool
	var isCanary bool

	cmdFlags := flag.NewFlagSet("create", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	cmdFlags.BoolVar(&isEmergency, "emergency", false, "emergency")
	cmdFlags.BoolVar(&isCanary, "canary", false, "canary")
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("%v", err))
		return 1
	}

	environment := "prod"
	if isCanary {
		environment = "canary"
	}

	c.Ui.Output(fmt.Sprintf("Creating LC for %s environment.\n", environment))

	appSelector := core.NewCliAppSelector(c.Ui)
	selectedApp, err := appSelector.Choose(c.Apps)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	var params *autoscaling.DescribeAccountLimitsInput
	accountLimits, err := c.AsgService.DescribeAccountLimits(params)

	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	maxLCs := *accountLimits.MaxNumberOfLaunchConfigurations

	lcParams := &autoscaling.DescribeLaunchConfigurationsInput{
		MaxRecords: aws.Int64(100),
	}

	currentLCCount := 0
	err = c.AsgService.DescribeLaunchConfigurationsPages(lcParams,
		func(page *autoscaling.DescribeLaunchConfigurationsOutput, lastPage bool) bool {
			currentLCCount += len(page.LaunchConfigurations)
			return !lastPage
		})

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Info(fmt.Sprintf("Current Launch Config Capacity: %d/%d", currentLCCount, maxLCs))

	selectedAmi, err := c.AmiSelector.Select(*selectedApp, environment)

	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Info(fmt.Sprintf("You selected %s\n", selectedAmi.Name))
	c.Ui.Info(fmt.Sprintf("Version Number: %s\n", selectedAmi.Version))

	emergencyText := ""
	if isEmergency {
		emergencyText = "-emergency"
	}

	launchConfigName := fmt.Sprintf("%s-%s-%s%s", selectedApp.Name, environment, selectedAmi.Version, emergencyText)

	//Create deployment-specific KeyPair

	keyName := fmt.Sprintf("%s-%d", launchConfigName, time.Now().Unix())

	keyUploadResults, err := c.KeyService.Upload(keyName, *selectedApp, environment)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Info(fmt.Sprintf("Created KeyPair: %s. \n", keyUploadResults.KeyName))

	createLCParams := &autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName:  aws.String(launchConfigName), // Required
		AssociatePublicIpAddress: aws.Bool(true),
		IamInstanceProfile:       aws.String(selectedApp.InstanceProfile),
		ImageId:                  aws.String(selectedAmi.Id),
		InstanceMonitoring: &autoscaling.InstanceMonitoring{
			Enabled: aws.Bool(true),
		},
		InstanceType: aws.String(selectedApp.InstanceType),
		KeyName:      aws.String(keyName),
		SecurityGroups: []*string{
			aws.String(selectedApp.SecurityGroup), // Required
		},
		UserData: aws.String(selectedAmi.UserData),
	}

	deployAction := NewDeployAction("create", selectedApp.Name, launchConfigName, 0)

	c.Ui.Info(fmt.Sprint("Creating Launch Configuration with the following parameters:"))
	c.Ui.Info(fmt.Sprint(createLCParams))
	ok, err := c.Ui.Ask("'ok' if you agree, anything else to cancel: ")
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		c.Cleanup(keyUploadResults)
		return 1
	}

	if ok != "ok" {
		c.Ui.Warn("Cancelled.")
		if !c.Cleanup(keyUploadResults) {
			return 1
		}
		return 0
	}

	_, createError := c.AsgService.CreateLaunchConfiguration(createLCParams)

	if createError != nil {
		// Message from an error.
		c.Ui.Error(fmt.Sprintf("Failed to create Launch Configuration: %s", launchConfigName))
		c.Ui.Error(fmt.Sprintln(createError.Error()))
		c.Cleanup(keyUploadResults)
		return 1
	}

	c.Notifier.Notify(deployAction)
	c.Ui.Output(fmt.Sprintln("Launch Configuration created."))

	return 0
}

func (c *CreateCommand) Cleanup(uploadRes *core.KeyUploadResult) bool {

	c.Ui.Info("")
	c.Ui.Info(fmt.Sprintf("Cleaning up created KeyPair: %s", uploadRes.KeyName))

	err := c.KeyService.CleanUp(uploadRes)
	if err != nil {
		c.Ui.Error(err.Error())
		return false
	}

	c.Ui.Info(fmt.Sprintf("Successfully deleted S3 object: %s", uploadRes.Key))

	return true
}

func (c *CreateCommand) Synopsis() string {
	return "Creates a launch configuration based on selected parameters."
}

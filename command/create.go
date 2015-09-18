package command

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/mitchellh/cli"
	"strings"
	"strconv"
	"github.com/aws/aws-sdk-go/service/ec2"
	"sort"
)

type suripuApp struct {
	name string
	sg string
	instanceType string
	instanceProfile string
	targetDesiredCapacity int64 //This is the desired capacity of the asg targeted for deployment
}

type ByImageTime []*ec2.Image

func (s ByImageTime) Len() int {
	return len(s)
}
func (s ByImageTime) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByImageTime) Less(i, j int) bool {
	return *s[i].CreationDate < *s[j].CreationDate
}

type CreateCommand struct {
	Ui cli.ColoredUi
}

func (c *CreateCommand) Help() string {
	helpText := `Usage: create`
	return strings.TrimSpace(helpText)
}

func (c *CreateCommand) Run(args []string) int {
	config := &aws.Config{
		Region: aws.String("us-east-1"),
	}
	asgService := autoscaling.New(config)
	ec2Service := ec2.New(config)

	c.Ui.Output("Which app are we building for?")

	suripuApps := []suripuApp{
		suripuApp{name: "suripu-app", sg: "sg-d28624b6", instanceType: "m3.medium", instanceProfile: "suripu-app"},
		suripuApp{name: "suripu-service", sg: "sg-11ac0e75", instanceType: "m3.medium", instanceProfile: "suripu-service"},
		suripuApp{name: "suripu-workers", sg: "sg-7054d714", instanceType: "c3.xlarge", instanceProfile: "suripu-workers"},
		suripuApp{name: "suripu-admin", sg: "sg-71773a16", instanceType: "t2.micro", instanceProfile: "suripu-admin"},
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

	selectedApp := suripuApps[appIdx]

	var params *autoscaling.DescribeAccountLimitsInput
	accountLimits, err := asgService.DescribeAccountLimits(params)

	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	maxLCs := *accountLimits.MaxNumberOfLaunchConfigurations

	lcParams := &autoscaling.DescribeLaunchConfigurationsInput{
		MaxRecords: aws.Int64(100),
	}
	descLCs, err := asgService.DescribeLaunchConfigurations(lcParams)
	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		c.Ui.Error(err.Error())
		return 1
	}

	currentLCCount := len(descLCs.LaunchConfigurations)
	c.Ui.Info(fmt.Sprintf("Current Launch Config Capacity: %d/%d", currentLCCount, maxLCs))

	//Grab latest 5 boxfuse-created AMIs
	amiTagName := fmt.Sprintf("timbart/%s", strings.Replace(selectedApp.name, "-", "", -1))

	ec2Params := &ec2.DescribeImagesInput{
		DryRun: aws.Bool(false),
		ExecutableUsers: []*string{
			aws.String("self"), // Required
			// More values...
		},
		Filters: []*ec2.Filter{
			{ // Required
				Name: aws.String("tag-value"),
				Values: []*string{
					aws.String(amiTagName), // Required
					// More values...
				},
			},
			{ // Required
				Name: aws.String("tag-key"),
				Values: []*string{
					aws.String("boxfuse:app"), // Required
					// More values...
				},
			},
		},
	}
	resp, err := ec2Service.DescribeImages(ec2Params)

	if err != nil {
		c.Ui.Error(fmt.Sprintln(err.Error()))
		return 1
	}

	amiId := ""
	amiName := ""
	amiVersion := ""

	if len(resp.Images) > 0 {
	sort.Sort(sort.Reverse(ByImageTime(resp.Images)))

	c.Ui.Output("Which AMI should be used?")
	numImages := Min(len(resp.Images), 5)
	for idx := 0; idx < numImages; idx++ {
		c.Ui.Output(fmt.Sprintf("[%d]\t%s\t%s", idx, *resp.Images[idx].Name, *resp.Images[idx].CreationDate))
	}

	ami, err := c.Ui.Ask("Select an AMI #: ")
	amiIdx, _ := strconv.Atoi(ami)

	if err != nil || amiIdx >= len(resp.Images) {
		c.Ui.Error(fmt.Sprintf("Incorrect AMI selection: %s\n", err))
		return 1
	}

	amiName = *resp.Images[amiIdx].Name
	amiId = *resp.Images[amiIdx].ImageId
	//Parse out version number
	amiNameInfo := strings.Split(amiName, "_")
	amiVersion = amiNameInfo[3]

	} else {
		//Allow user to enter version number and search for AMI based on that
		c.Ui.Warn(fmt.Sprintf("No Boxfuse AMIs found for %s. Proceeding with Packer-created AMI selection.", selectedApp.name))

		ec2ParamsAll := &ec2.DescribeImagesInput{
			DryRun: aws.Bool(false),
			Filters: []*ec2.Filter{
				{ // Required
					Name: aws.String("is-public"),
					Values: []*string{
						aws.String("false"), // Required
						// More values...
					},
				},
			},
		}
		respAll, err := ec2Service.DescribeImages(ec2ParamsAll)

		if err != nil {
			c.Ui.Error(fmt.Sprintln(err.Error()))
			return 1
		}

		validImages := make([]*ec2.Image, 0)

		for _, image := range respAll.Images {
			if strings.HasPrefix(*image.Name, selectedApp.name) {
				validImages = append(validImages, image)
			}
		}

		sort.Sort(sort.Reverse(ByImageTime(validImages)))

		c.Ui.Output("Which AMI should be used?")
		numImages := Min(len(validImages), 10)
		for idx := 0; idx < numImages; idx++ {
			c.Ui.Output(fmt.Sprintf("[%d] \t%s\t%s", idx, *validImages[idx].Name, *validImages[idx].CreationDate))
		}

		ami, err := c.Ui.Ask("Select an AMI #: ")
		amiIdx, _ := strconv.Atoi(ami)

		if err != nil || amiIdx >= len(validImages) {
			c.Ui.Error(fmt.Sprintf("Incorrect AMI selection: %s\n", err))
			return 1
		}

		amiName = *validImages[amiIdx].Name
		amiId = *validImages[amiIdx].ImageId
		//Parse out version number
		amiNameInfo := strings.Split(amiName, "-")
		amiVersion = amiNameInfo[2]
	}

	c.Ui.Info(fmt.Sprintf("You selected %s\n", amiName))
	c.Ui.Info(fmt.Sprintf("Version Number: %s\n", amiVersion))

	launchConfigName := fmt.Sprintf("%s-prod-%s", selectedApp.name, amiVersion)

	createLCParams := &autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName:  aws.String(launchConfigName), // Required
		AssociatePublicIpAddress: aws.Bool(true),
		IamInstanceProfile: aws.String(selectedApp.instanceProfile),
		ImageId:            aws.String(amiId),
		InstanceMonitoring: &autoscaling.InstanceMonitoring{
			Enabled: aws.Bool(true),
		},
		InstanceType:     aws.String(selectedApp.instanceType),
		KeyName:          aws.String("vpc-prod"),
		SecurityGroups: []*string{
			aws.String(selectedApp.sg), // Required
		},
		//UserData:  aws.String("XmlStringUserData"),
	}

	c.Ui.Info(fmt.Sprint("Creating Launch Configuration with the following parameters:"))
	c.Ui.Info(fmt.Sprint(createLCParams))
	ok, err := c.Ui.Ask("'ok' if you agree, anything else to cancel: ")
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	if ok != "ok" {
		c.Ui.Warn("Cancelled.")
		return 0
	}

	_, createError := asgService.CreateLaunchConfiguration(createLCParams)

	if createError != nil {
		// Message from an error.
		c.Ui.Error(fmt.Sprintf("Failed to create Launch Configuration: %s", launchConfigName))
		c.Ui.Error(fmt.Sprintln(createError.Error()))
		return 1
	}

	c.Ui.Output(fmt.Sprintln("Launch Configuration created."))

	return 0
}

func (c *CreateCommand) Synopsis() string {
	return "Creates a launch configuration based on selected parameters. (Only for boxfuse-created AMIs)"
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
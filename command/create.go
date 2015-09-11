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
	"time"
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


	var params *autoscaling.DescribeAccountLimitsInput
	accountLimits, err := asgService.DescribeAccountLimits(params)

	if err != nil {
		fmt.Println(err.Error())
		return 1
	}
	maxLCs := *accountLimits.MaxNumberOfLaunchConfigurations
	descLCs, err := asgService.DescribeLaunchConfigurations(&autoscaling.DescribeLaunchConfigurationsInput{})
	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return 1
	}

	currentLCCount := len(descLCs.LaunchConfigurations)
	fmt.Sprintf("Current LC Capacity: %d/%d", currentLCCount, maxLCs)


	//Grab latest AMI for specified version of specified app
	amiTagName := fmt.Sprintf("timbart/%s", strings.Replace(suripuApps[appIdx].name, "-", "", -1))

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
		fmt.Println(err.Error())
		return 1
	}

	sort.Sort(sort.Reverse(ByImageTime(resp.Images)))

	c.Ui.Output("Which AMI should be used?")
	numImages := Min(len(resp.Images), 4)
	for idx := 0; idx < numImages; idx++ {
		fmt.Printf("[%d]\t%s\t%s\n", idx, *resp.Images[idx].Name, *resp.Images[idx].CreationDate)
	}

	ami, err := c.Ui.Ask("Select an AMI #: ")
	amiIdx, _ := strconv.Atoi(ami)

	if err != nil || amiIdx >= len(resp.Images) {
		c.Ui.Error(fmt.Sprintf("Incorrect AMI selection: %s\n", err))
		return 1
	}

	amiName := *resp.Images[amiIdx].Name
	amiId := *resp.Images[amiIdx].ImageId
	fmt.Printf("You selected %s\n", amiName)

	//Parse out version number
	amiNameInfo := strings.Split(amiName, "_")
	amiVersion := amiNameInfo[3]
	fmt.Printf("Version Number: %s\n", amiVersion)

	timestamp := time.Now().Unix()
	launchConfigName := fmt.Sprintf("%s-prod-%s-%d", suripuApps[appIdx].name, amiVersion, timestamp)

	createLCParams := &autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName:  aws.String(launchConfigName), // Required
		AssociatePublicIpAddress: aws.Bool(true),
		IamInstanceProfile: aws.String(suripuApps[appIdx].instanceProfile),
		ImageId:            aws.String(amiId),
		InstanceMonitoring: &autoscaling.InstanceMonitoring{
			Enabled: aws.Bool(true),
		},
		InstanceType:     aws.String(suripuApps[appIdx].instanceType),
		KeyName:          aws.String("vpc-prod"),
		SecurityGroups: []*string{
			aws.String(suripuApps[appIdx].sg), // Required
		},
		//UserData:  aws.String("XmlStringUserData"),
	}

	fmt.Println(createLCParams)
	ok, err := c.Ui.Ask("'ok' if you agree, anything else to cancel: ")
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	if ok != "ok" {
		c.Ui.Warn("Cancelled.")
		return 0
	}

	_, ferr := asgService.CreateLaunchConfiguration(createLCParams)

	if ferr != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Printf("Failed to create Launch Configuration: %s", launchConfigName)
		fmt.Println(ferr.Error())
		return 1
	}

	fmt.Println("Launch Configuration created.")

	return 0
}

func (c *CreateCommand) Synopsis() string {
	return "Creates a launch configuration for the specified application at specified version#."
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
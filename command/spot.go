package command

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hello/sanders/shared"
	"github.com/mitchellh/cli"
	"strings"
)

//This hash should be updated anytime default_userdata.sh is updated on S3
// var expectedUserDataHash = "0011ed8a3aeaffa830620d16e39f84549cb0c6cb"

type SpotCommand struct {
	Ui                cli.ColoredUi
	Notifier          BasicNotifier
	UserDataGenerator *shared.UserMetaDataGenerator
}

func (c *SpotCommand) Help() string {
	helpText := `Usage: spot`
	return strings.TrimSpace(helpText)
}

type Runner interface {
	Run() int
}

type FleetDescriber struct {
	Ui  cli.ColoredUi
	srv *ec2.EC2
}

func (f *FleetDescriber) Run() int {
	describe := &ec2.DescribeSpotFleetRequestsInput{}

	result, err := f.srv.DescribeSpotFleetRequests(describe)
	if err != nil {
		f.Ui.Error(err.Error())
		return 1
	}

	for _, config := range result.SpotFleetRequestConfigs {
		row := fmt.Sprintf("%s\t%s", *config.SpotFleetRequestState, *config.SpotFleetRequestId)
		if *config.SpotFleetRequestState == "active" {

			f.Ui.Info(row)
			f.Ui.Info("\t" + fmt.Sprintf("%s: %0.f/%d", *config.SpotFleetRequestConfig.AllocationStrategy, *config.SpotFleetRequestConfig.FulfilledCapacity, *config.SpotFleetRequestConfig.TargetCapacity))
			f.Ui.Info("")

			describeTags := &ec2.DescribeTagsInput{
				Filters: []*ec2.Filter{

					// &ec2.Filter{
					// 	Name:   aws.String("tag:aws:ec2spot:fleet-request-id"),
					// 	Values: aws.StringSlice([]string{*config.SpotFleetRequestId}),
					// },
					&ec2.Filter{
						Name:   aws.String("SpotFleetRequestId"),
						Values: aws.StringSlice([]string{*config.SpotFleetRequestId}),
					},
				},
			}
			tags, err := f.srv.DescribeTags(describeTags)
			if err != nil {
				f.Ui.Error(err.Error())
				return 1
			}
			f.Ui.Info(fmt.Sprintf("tags: %d", len(tags.Tags)))
			for _, tag := range tags.Tags {
				f.Ui.Info(fmt.Sprintf("%s %s:%s", *tag.ResourceId, *tag.Key, *tag.Value))
			}
		}

	}

	return 0
}

type FleetCanceller struct {
	Ui cli.ColoredUi
}

type FleetCreator struct {
	Ui                cli.ColoredUi
	srv               *ec2.EC2
	userDataGenerator *shared.UserMetaDataGenerator
}

func (f *FleetCreator) Run() int {

	specs := []*ec2.SpotFleetLaunchSpecification{}

	subnets := []string{"subnet-28c6565f", "subnet-da02b383"}

	metadataInput := &shared.UserMetaDataInput{
		AmiVersion:    "0.7.334",
		AppName:       "suripu-workers",
		PackagePath:   "com/hello/suripu",
		CanaryPath:    "",
		DefaultRegion: "us-east-1",
		JavaVersion:   8,
	}
	userData, err := f.userDataGenerator.Parse(metadataInput)

	if err != nil {
		f.Ui.Error(err.Error())
		return 1
	}

	for _, subnet := range subnets {
		launchSpec := &ec2.SpotFleetLaunchSpecification{
			ImageId: aws.String("ami-16d5ee01"),
			IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
				Arn: aws.String("arn:aws:iam::053216739513:instance-profile/suripu-workers"),
			},
			InstanceType: aws.String("c3.xlarge"),
			KeyName:      aws.String("vpc-prod"),
			SpotPrice:    aws.String("0.210"),
			SubnetId:     aws.String(subnet),
			UserData:     aws.String(userData),
			SecurityGroups: []*ec2.GroupIdentifier{
				&ec2.GroupIdentifier{
					GroupId: aws.String("sg-7054d714"),
				},
			},
		}
		specs = append(specs, launchSpec)
	}

	configData := &ec2.SpotFleetRequestConfigData{
		SpotPrice:            aws.String("0.210"),
		AllocationStrategy:   aws.String("diversified"),
		IamFleetRole:         aws.String("arn:aws:iam::053216739513:role/ec2-spot-fleet"),
		TargetCapacity:       aws.Int64(2),
		LaunchSpecifications: specs,
	}

	input := &ec2.RequestSpotFleetInput{
		SpotFleetRequestConfig: configData,
	}

	output, err := f.srv.RequestSpotFleet(input)
	if err != nil {
		f.Ui.Error(err.Error())
		return 1
	}

	f.Ui.Info(fmt.Sprintf("request-id: %s", *output.SpotFleetRequestId))
	return 0
}

func (c *SpotCommand) Run(args []string) int {

	var cmd string

	cmdFlags := flag.NewFlagSet("spot", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }
	cmdFlags.StringVar(&cmd, "cmd", "", "")

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	if cmd == "" {
		c.Ui.Error("Missing command")
		return 1
	}

	config := &aws.Config{
		Region: aws.String("us-east-1"),
	}

	ec2Service := ec2.New(session.New(), config)

	var runner Runner
	switch cmd {
	case "status":
		runner = &FleetDescriber{
			Ui:  c.Ui,
			srv: ec2Service,
		}
	case "create":
		runner = &FleetCreator{
			Ui:                c.Ui,
			srv:               ec2Service,
			userDataGenerator: c.UserDataGenerator,
		}
	default:
		c.Ui.Error(fmt.Sprintf("Unknown command: %s", cmd))
		return 1
	}

	return runner.Run()
}

func (c *SpotCommand) Synopsis() string {
	return "Creates a launch configuration based on selected parameters."
}

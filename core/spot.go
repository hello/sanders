package core

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mitchellh/cli"
)

type FleetManager struct {
	ui  cli.ColoredUi
	srv *ec2.EC2
}

func NewFleetManager(ui cli.ColoredUi, srv *ec2.EC2) *FleetManager {
	return &FleetManager{
		ui:  ui,
		srv: srv,
	}
}

func (f *FleetManager) Describe() error {
	describe := &ec2.DescribeSpotFleetRequestsInput{}

	result, err := f.srv.DescribeSpotFleetRequests(describe)

	if err != nil {
		return err
	}
	for _, config := range result.SpotFleetRequestConfigs {
		status := "fullfilled"
		if config.ActivityStatus != nil {
			status = *config.ActivityStatus
		}

		if *config.SpotFleetRequestState == "active" || *config.SpotFleetRequestState == "submitted" {
			f.ui.Info(fmt.Sprintf("%s", *config.SpotFleetRequestId))
			f.ui.Info(fmt.Sprintf("\tstatus: %s", status))
			f.ui.Info(fmt.Sprintf("\tsettings: %s: %0.f/%d", *config.SpotFleetRequestConfig.AllocationStrategy, *config.SpotFleetRequestConfig.FulfilledCapacity, *config.SpotFleetRequestConfig.TargetCapacity))
			f.ui.Info(fmt.Sprintf("\tcreated: %s", *config.CreateTime))

			input := &ec2.DescribeSpotFleetInstancesInput{
				SpotFleetRequestId: config.SpotFleetRequestId,
			}

			output, err := f.srv.DescribeSpotFleetInstances(input)
			if err != nil {
				f.ui.Error(err.Error())
				return err
			}
			instanceIds := make([]string, 0)
			spotInstanceReqIds := make([]string, 0)
			for _, instance := range output.ActiveInstances {
				instanceIds = append(instanceIds, *instance.InstanceId)
				spotInstanceReqIds = append(spotInstanceReqIds, *instance.SpotInstanceRequestId)
			}

			if len(instanceIds) == 0 {
				f.ui.Warn("\tNot ready yet: " + status)
				f.ui.Output("")
				continue
			}

			spotIds := &ec2.DescribeSpotInstanceRequestsInput{
				SpotInstanceRequestIds: aws.StringSlice(spotInstanceReqIds),
			}

			individualQueries, err := f.srv.DescribeSpotInstanceRequests(spotIds)
			if err != nil {
				return err
			}

			for _, spotReq := range individualQueries.SpotInstanceRequests {
				f.ui.Info(fmt.Sprintf("\tinstance-id: %s", *spotReq.InstanceId))
				f.ui.Info(fmt.Sprintf("\t\t-price: %s", *spotReq.SpotPrice))
				f.ui.Info(fmt.Sprintf("\t\t-az: %s", *spotReq.LaunchedAvailabilityZone))
				f.ui.Info(fmt.Sprintf("\t\t-created: %s", *spotReq.CreateTime))

			}

			describeInstancesInput := &ec2.DescribeInstancesInput{
				InstanceIds: aws.StringSlice(instanceIds),
			}
			out, err := f.srv.DescribeInstances(describeInstancesInput)
			if err != nil {
				f.ui.Error(err.Error())
				return err
			}

			for _, res := range out.Reservations {
				for _, instance := range res.Instances {
					f.ui.Info(fmt.Sprintf("\t%s, %s", *instance.InstanceId, *instance.KeyName))
				}
			}
			f.ui.Output("")
		}
	}

	return nil
}

func (f *FleetManager) Execute(configData *ec2.SpotFleetRequestConfigData) (string, error) {
	input := &ec2.RequestSpotFleetInput{
		SpotFleetRequestConfig: configData,
	}

	output, err := f.srv.RequestSpotFleet(input)
	if err != nil {
		return "", err
	}

	f.ui.Info(fmt.Sprintf("request-id: %s", *output.SpotFleetRequestId))
	return *output.SpotFleetRequestId, err
}

func (f *FleetManager) Create(app *SuripuApp, ami *SelectedAmi, keyName string) (*ec2.SpotFleetRequestConfigData, error) {

	if app.Spot == nil {
		return nil, errors.New("Not configured for spot")
	}

	specs := []*ec2.SpotFleetLaunchSpecification{}

	subnets := []string{"subnet-28c6565f", "subnet-da02b383"}

	for _, subnet := range subnets {
		launchSpec := &ec2.SpotFleetLaunchSpecification{
			ImageId: aws.String(ami.Id),
			IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
				Arn: aws.String("arn:aws:iam::053216739513:instance-profile/" + app.InstanceProfile),
			},
			InstanceType: aws.String(app.InstanceType),
			KeyName:      aws.String(keyName),
			SpotPrice:    aws.String(app.Spot.Price),
			SubnetId:     aws.String(subnet),
			UserData:     aws.String(ami.UserData),
			SecurityGroups: []*ec2.GroupIdentifier{
				&ec2.GroupIdentifier{
					GroupId: aws.String(app.SecurityGroup),
				},
			},
		}
		specs = append(specs, launchSpec)
	}

	configData := &ec2.SpotFleetRequestConfigData{
		SpotPrice:            aws.String(app.Spot.Price),
		AllocationStrategy:   aws.String("diversified"),
		IamFleetRole:         aws.String("arn:aws:iam::053216739513:role/ec2-spot-fleet"),
		TargetCapacity:       aws.Int64(app.TargetDesiredCapacity),
		LaunchSpecifications: specs,
	}

	// TAG

	return configData, nil
}

func (f *FleetManager) Cancel(requestId string) error {

	input := &ec2.CancelSpotFleetRequestsInput{
		SpotFleetRequestIds: aws.StringSlice([]string{requestId}),
		TerminateInstances:  aws.Bool(true),
	}
	out, err := f.srv.CancelSpotFleetRequests(input)
	if err != nil {
		return err
	}

	f.ui.Info("status:")
	for _, req := range out.SuccessfulFleetRequests {
		f.ui.Info("\tsuccess: " + *req.SpotFleetRequestId)
	}
	for _, req := range out.UnsuccessfulFleetRequests {
		f.ui.Warn("\tfailed: " + *req.SpotFleetRequestId)
	}
	return nil
}

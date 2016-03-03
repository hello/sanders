package command

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/mitchellh/cli"
	"strings"
)

type StatusCommand struct {
	Ui       cli.ColoredUi
	Notifier BasicNotifier
}

func (c *StatusCommand) Help() string {
	helpText := `Usage: sanders status`
	return strings.TrimSpace(helpText)
}

type Status struct {
	ElbName  string
	Statuses []HostStatus
	Error    error
}

type HostStatus struct {
	Hostname    string
	Version     string
	InstanceId  string
	State       string
	Reason      string
	Description string
	Launched    string
}

func fetch(elbName string, service *elb.ELB, ec2Service *ec2.EC2, statuses chan *Status) {
	statuses <- elbStatus(elbName, service, ec2Service)
}

func (c *StatusCommand) Run(args []string) int {

	config := &aws.Config{
		Region: aws.String("us-east-1"),
	}

	service := elb.New(session.New(), config)
	ec2Service := ec2.New(session.New(), config)

	elbs := []string{
		"suripu-service-prod",
		"suripu-app-prod",
		"suripu-app-canary",
		"suripu-admin-prod",
		"messeji-prod",
		"hello-time-prod",
	}

	statuses := make(chan *Status, 0)

	for _, elbName := range elbs {
		go fetch(elbName, service, ec2Service, statuses)
		c.Ui.Info(fmt.Sprintf("Fetching: ELB %s", elbName))
	}

	c.Ui.Output("")
	results := make(map[string]*Status)

	remaining := len(elbs)
	for status := range statuses {
		if status.Error != nil {
			c.Ui.Error(fmt.Sprintf("%s", status.Error))
			return 1
		}
		results[status.ElbName] = status

		remaining -= 1
		if remaining == 0 {
			break
		}
	}

	for _, elb := range elbs {
		status, _ := results[elb]
		c.printStatus(status)
	}

	close(statuses)
	c.Ui.Output("")
	return 0
}

func elbStatus(elbName string, service *elb.ELB, ec2Service *ec2.EC2) *Status {
	req := &elb.DescribeInstanceHealthInput{
		LoadBalancerName: &elbName,
	}

	statuses := make([]HostStatus, 0)

	status := &Status{
		ElbName:  elbName,
		Statuses: statuses,
		Error:    nil,
	}

	lbResp, err := service.DescribeInstanceHealth(req)
	if err != nil {
		status.Error = err
		return status
	}

	instanceIds := make([]*string, 0)

	for _, state := range lbResp.InstanceStates {
		instanceIds = append(instanceIds, state.InstanceId)
	}

	instanceReq := &ec2.DescribeInstancesInput{
		InstanceIds: instanceIds,
	}

	resp, _ := ec2Service.DescribeInstances(instanceReq)

	publicNames := make(map[string]string, 0)
	amis := make(map[string]string, 0)
	amisNames := make(map[string]string, 0)
	amisToFetch := make([]*string, 0)
	instanceLaunchTimes := make(map[string]string, 0)
	lcNames := make(map[string]string, 0)

	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			publicNames[*instance.InstanceId] = *instance.PublicDnsName
			amis[*instance.InstanceId] = *instance.ImageId
			instanceLaunchTimes[*instance.InstanceId] = fmt.Sprintf("%s", *instance.LaunchTime)
			amisToFetch = append(amisToFetch, instance.ImageId)
			for _, tag := range instance.Tags {
				if strings.Contains(*tag.Key, "Launch Configuration") {
					lcNames[*instance.InstanceId] = *tag.Value
				}
			}
		}
	}

	amiReq := &ec2.DescribeImagesInput{
		ImageIds: amisToFetch,
	}

	amiResp, _ := ec2Service.DescribeImages(amiReq)
	for _, ami := range amiResp.Images {
		amisNames[*ami.ImageId] = *ami.Name
	}

	for _, state := range lbResp.InstanceStates {
		res, _ := publicNames[*state.InstanceId]
		amiId, _ := amis[*state.InstanceId]
		amiName, _ := amisNames[amiId]
		launchTime, _ := instanceLaunchTimes[*state.InstanceId]

		parts := make([]string, 0)
		imageVersion := ""
		if lcNames[*state.InstanceId] != "" {
			parts = strings.SplitAfterN(lcNames[*state.InstanceId], "-", 4)
			imageVersion = parts[len(parts)-1]
		} else {
			parts = strings.SplitAfterN(amiName, "-", 4)
			imageVersion = parts[2]
		}

		hostStatus := HostStatus{
			Version:     strings.TrimSuffix(imageVersion, "-"),
			InstanceId:  *state.InstanceId,
			State:       *state.State,
			Launched:    launchTime,
			Description: *state.Description,
			Reason:      *state.ReasonCode,
			Hostname:    res,
		}
		status.Statuses = append(status.Statuses, hostStatus)
	}

	return status
}

func (c *StatusCommand) printStatus(status *Status) {
	c.Ui.Info(status.ElbName)
	for _, status := range status.Statuses {

		if status.State == "InService" {
			c.Ui.Info(fmt.Sprintf("\tVersion: %s", status.Version))
			c.Ui.Info(fmt.Sprintf("\tID: %s", status.InstanceId))
			c.Ui.Info(fmt.Sprintf("\tState: %s", status.State))
			c.Ui.Info(fmt.Sprintf("\tLaunched: %s", status.Launched))
			c.Ui.Info(fmt.Sprintf("\tHostname: %s", status.Hostname))

		} else if status.Reason == "Instance is in pending state" {
			c.Ui.Warn(fmt.Sprintf("\tVersion: %s", status.Version))
			c.Ui.Warn(fmt.Sprintf("\tID: %s", status.InstanceId))
			c.Ui.Warn(fmt.Sprintf("\tState: %s", status.State))
			c.Ui.Warn(fmt.Sprintf("\tReason: %s", status.Reason))
			c.Ui.Warn(fmt.Sprintf("\tDescription: %s", status.Description))
			c.Ui.Warn(fmt.Sprintf("\tLaunched: %s", status.Launched))
			c.Ui.Warn(fmt.Sprintf("\tHostname: %s", status.Hostname))
		} else {
			c.Ui.Error(fmt.Sprintf("\tVersion: %s", status.Version))
			c.Ui.Error(fmt.Sprintf("\tID: %s", status.InstanceId))
			c.Ui.Error(fmt.Sprintf("\tState: %s", status.State))
			c.Ui.Error(fmt.Sprintf("\tReason: %s", status.Reason))
			c.Ui.Error(fmt.Sprintf("\tDescription: %s", status.Description))
			c.Ui.Error(fmt.Sprintf("\tLaunched: %s", status.Launched))
			c.Ui.Error(fmt.Sprintf("\tHostname: %s", status.Hostname))
		}
		c.Ui.Output("")
	}
}

func (c *StatusCommand) Synopsis() string {
	return "See ELB status"
}

package command

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mitchellh/cli"
	"io/ioutil"
	"os"
	"strings"
)

type HostsCommand struct {
	Ui       cli.ColoredUi
	Notifier BasicNotifier
}

func (c *HostsCommand) Help() string {
	helpText := `Usage: sanders hosts [-nosync]`
	return strings.TrimSpace(helpText)
}

func (c *HostsCommand) Run(args []string) int {

	cmdFlags := flag.NewFlagSet("hosts", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }
	var nosync = cmdFlags.Bool("nosync", false, "disable syncing dsh groupnames")
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	apps := []string{"suripu-app", "suripu-service", "suripu-workers", "sense-firehose", "suripu-queue"}

	config := &aws.Config{
		Region: aws.String("us-east-1"),
	}

	service := autoscaling.New(session.New(), config)
	ec2Service := ec2.New(session.New(), config)

	groupnames := make([]*string, 0)
	for _, appName := range apps {
		one := fmt.Sprintf("%s-prod", appName)
		two := fmt.Sprintf("%s-prod-green", appName)
		groupnames = append(groupnames, &one)
		groupnames = append(groupnames, &two)
		// groupnames = append(groupnames, "suripu-prdo")
	}

	req := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: groupnames,
	}

	resp, _ := service.DescribeAutoScalingGroups(req)

	for _, asg := range resp.AutoScalingGroups {
		instanceIds := make([]*string, 0)

		for _, instance := range asg.Instances {
			instanceIds = append(instanceIds, instance.InstanceId)
		}

		if len(instanceIds) == 0 {
			c.Ui.Warn(fmt.Sprintf("No instance for ASG: %s", *asg.AutoScalingGroupName))
			continue
		}

		c.Ui.Info(fmt.Sprintf("ASG: %s [%s]", *asg.AutoScalingGroupName, *asg.LaunchConfigurationName))

		describeReq := &ec2.DescribeInstancesInput{
			InstanceIds: instanceIds,
		}

		describeResp, err := ec2Service.DescribeInstances(describeReq)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("%s", err))
			return 1
		}

		content := ""
		for _, reservation := range describeResp.Reservations {
			for _, instance := range reservation.Instances {
				content += fmt.Sprintf("%s\n", *instance.PublicDnsName)
				c.Ui.Error(fmt.Sprintf("\t%s", *instance.PublicDnsName))
			}
		}
		if !*nosync {
			homedir := os.Getenv("HOME")
			filePath := homedir + "/.dsh/group/" + *asg.AutoScalingGroupName
			err = ioutil.WriteFile(filePath, []byte(content), 0644)
			if err != nil {
				c.Ui.Info(fmt.Sprintf("Failed saving file %s. %s", *asg.AutoScalingGroupName, err))
			}
			c.Ui.Output(fmt.Sprintf("Saved to :%s", filePath))
		}

		c.Ui.Info("")
	}

	c.Ui.Output("")
	return 0
}

func (c *HostsCommand) Synopsis() string {
	return "Lists PublicDNSName for all instances in all ASGs"
}

package command

import (
	"flag"
	"fmt"
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/autoscaling"
	"github.com/awslabs/aws-sdk-go/gen/ec2"
	"github.com/mitchellh/cli"
	"io/ioutil"
	"strings"
)

type HostsCommand struct {
	Ui cli.ColoredUi
}

func (c *HostsCommand) Help() string {
	helpText := `Usage: sanders hosts [-sync]`
	return strings.TrimSpace(helpText)
}

func (c *HostsCommand) Run(args []string) int {

	cmdFlags := flag.NewFlagSet("hosts", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }
	var sync = cmdFlags.Bool("sync", false, "syncs dsh groupnames")
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	apps := []string{"suripu-app", "suripu-service", "suripu-workers"}

	creds, _ := aws.EnvCreds()
	cli := autoscaling.New(creds, "us-east-1", nil)
	ec2Cli := ec2.New(creds, "us-east-1", nil)

	groupnames := make([]string, 0)
	for _, appName := range apps {
		groupnames = append(groupnames, fmt.Sprintf("%s-prod", appName))
		groupnames = append(groupnames, fmt.Sprintf("%s-prod-green", appName))
	}

	req := &autoscaling.AutoScalingGroupNamesType{
		AutoScalingGroupNames: groupnames,
	}

	resp, err := cli.DescribeAutoScalingGroups(req)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	for _, asg := range resp.AutoScalingGroups {
		instanceIds := make([]string, 0)

		for _, instance := range asg.Instances {
			instanceIds = append(instanceIds, *instance.InstanceID)
		}

		if len(instanceIds) == 0 {
			c.Ui.Warn(fmt.Sprintf("No instance for ASG: %s", *asg.AutoScalingGroupName))
			continue
		}

		c.Ui.Info(fmt.Sprintf("ASG: %s", *asg.AutoScalingGroupName))

		describeReq := &ec2.DescribeInstancesRequest{
			InstanceIDs: instanceIds,
		}

		describeResp, err := ec2Cli.DescribeInstances(describeReq)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("%s", err))
			return 1
		}

		content := ""
		for _, reservation := range describeResp.Reservations {
			for _, instance := range reservation.Instances {
				content += fmt.Sprintf("%s\n", *instance.PublicDNSName)
				c.Ui.Error(fmt.Sprintf("\t%s", *instance.PublicDNSName))
			}
		}
		if *sync {

			filePath := "/Users/tim/.dsh/group/" + *asg.AutoScalingGroupName
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

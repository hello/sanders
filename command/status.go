package command

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mitchellh/cli"
	"strings"
	"github.com/aws/aws-sdk-go/aws/session"
	"time"
	"sort"
	"flag"
)

type StatusCommand struct {
	Ui cli.ColoredUi
}

type ByInstanceTagLaunchConfiguration []*ec2.Instance

func (s ByInstanceTagLaunchConfiguration) Len() int {
	return len(s)
}
func (s ByInstanceTagLaunchConfiguration) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByInstanceTagLaunchConfiguration) Less(i, j int) bool {
	iLaunchConfig := ""
	jLaunchConfig := ""
	for _, iTag := range s[i].Tags {
		if *iTag.Key == "Launch Configuration" {
			iLaunchConfig = *iTag.Value
		}
	}
	for _, jTag := range s[j].Tags {
		if *jTag.Key == "Launch Configuration" {
			jLaunchConfig = *jTag.Value
		}
	}
	return iLaunchConfig < jLaunchConfig
}

type ByInstanceTagASG []*ec2.Instance

func (s ByInstanceTagASG) Len() int {
	return len(s)
}
func (s ByInstanceTagASG) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByInstanceTagASG) Less(i, j int) bool {
	iASG := ""
	jASG := ""
	for _, iTag := range s[i].Tags {
		if *iTag.Key == "aws:autoscaling:groupName" {
			iASG = *iTag.Value
		}
	}
	for _, jTag := range s[j].Tags {
		if *jTag.Key == "aws:autoscaling:groupName" {
			jASG = *jTag.Value
		}
	}
	return iASG < jASG
}

type ByInstanceLaunchTime []*ec2.Instance

func (s ByInstanceLaunchTime) Len() int {
	return len(s)
}
func (s ByInstanceLaunchTime) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByInstanceLaunchTime) Less(i, j int) bool {
	return s[i].LaunchTime.Unix() < s[j].LaunchTime.Unix()
}

func (c *StatusCommand) Help() string {
	helpText := `Usage: sanders status [-t]`
	return strings.TrimSpace(helpText)
}

func (c *StatusCommand) Run(args []string) int {

	cmdFlags := flag.NewFlagSet("status", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }
	var timeSort = cmdFlags.Bool("t", false, "Sort instances by launch time.")
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	config := &aws.Config{
		Region: aws.String("us-east-1"),
	}

	//service := elb.New(session.New(), config)
	ec2Service := ec2.New(session.New(), config)

	appNames := make([]*string, 0)
	for _, app := range suripuApps {
		appWilcard := fmt.Sprintf("%s-*", app.name)
		appNames = append(appNames, &appWilcard)
	}

	params := &ec2.DescribeInstancesInput{
		DryRun: aws.Bool(false),
		Filters: []*ec2.Filter{
			{ // Required
				Name: aws.String("tag:Launch Configuration"),
				Values: appNames,
			},
		},
	}
	resp, err := ec2Service.DescribeInstances(params)

	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}


	//Collect all instances from reservations
	instances := make([]*ec2.Instance, 0)
	for _, resv := range resp.Reservations {
		for _, inst := range resv.Instances {
			instances = append(instances, inst)
		}
	}

	//Sort instances
	if *timeSort {
		sort.Sort(ByInstanceLaunchTime(instances))
	} else {
		sort.Sort(ByInstanceTagASG(instances))
	}

	lastASG := ""
	asgInstanceCounts := make(map[string]int)
	c.Ui.Output(strings.Repeat("*", 140))
	c.Ui.Output(fmt.Sprintf("%-28s %-14s %-10s %-10s %-43s %-29s*", "Instance", "State", "ID", "Size", "Public DNS Name", "Launched At"))
	c.Ui.Output(strings.Repeat("*", 140))
	for _, inst := range instances {
		launchConfigName := ""
		for _, instTag := range inst.Tags {
			if *instTag.Key == "aws:autoscaling:groupName" {
				if lastASG != *instTag.Value {
					c.Ui.Info(fmt.Sprintf("ASG: %-25s %s", *instTag.Value, strings.Repeat("*", 109)))
					lastASG = *instTag.Value
				}
				asgInstanceCounts[*instTag.Value] = asgInstanceCounts[*instTag.Value] + 1
			}
			if *instTag.Key == "Launch Configuration" {
				launchConfigName = *instTag.Value
			}

		}

		c.Ui.Output(fmt.Sprintf("%-28s %-14s %-10s %-10s %-43s %-29s", launchConfigName, *inst.State.Name, *inst.InstanceId, *inst.InstanceType, *inst.PublicDnsName, inst.LaunchTime.Format(time.UnixDate)))
		if inst.StateReason != nil {
			c.Ui.Warn(fmt.Sprintf("%s", inst.StateReason.Message))
		}
	}

	c.Ui.Output(strings.Repeat("*", 140))

	for key, val := range asgInstanceCounts {
		for _, app := range suripuApps {
			if !strings.HasPrefix(key, app.name) {
				continue
			}

			if strings.Contains(key, "canary") {
				continue
			}

			if val < int(app.targetDesiredCapacity) {
				c.Ui.Warn(fmt.Sprintf("Warning: %s has only %d instance(s) running. Total desired capacity is: %d", key, val, app.targetDesiredCapacity))
				c.Ui.Warn(fmt.Sprintf("Please ensure you have fully deployed %s before proceeding.", app.name))
			}

			if val > int(app.targetDesiredCapacity) {
				c.Ui.Error(fmt.Sprintf("Error: %s has %d instance(s) running. This exceeds the total desired capacity of: %d", key, val, app.targetDesiredCapacity))
			}
		}
	}
	c.Ui.Output("")
	return 0
}

func (c *StatusCommand) Synopsis() string {
	return "See ELB status"
}

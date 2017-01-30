package command

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hello/sanders/core"
	"github.com/mitchellh/cli"
	"strings"
)

type TailCommand struct {
	Ui   cli.ColoredUi
	Apps []core.SuripuApp
	Srv  *ec2.EC2
}

func (c *TailCommand) Help() string {
	helpText := `Usage: sanders tail`
	return strings.TrimSpace(helpText)
}

func (c *TailCommand) Run(args []string) int {

	cmdFlags := flag.NewFlagSet("tail", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }
	var query = cmdFlags.String("query", "ERROR", "query to search in papertrail")
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	appSelector := core.NewCliAppSelector(c.Ui)
	selectedApp, err := appSelector.Choose(c.Apps)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("tag:Name"),
				Values: []*string{
					aws.String(selectedApp.Name + "-prod"),
				},
			},
		},
	}

	res, err := c.Srv.DescribeInstances(params)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	instances := make([]*ec2.Instance, 0)

	for _, r := range res.Reservations {
		for _, i := range r.Instances {
			instances = append(instances, i)
		}
	}

	selector := core.NewCliInstanceSelector(c.Ui)
	selected, err := selector.Choose(instances)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	// ip-0-0-0-0.ec2.internal -> ip-0-0-0-0
	dnsParts := strings.SplitN(*selected.PrivateDnsName, ".", 2)
	systemID := dnsParts[0]

	tailor := &core.Tailor{
		Ui: c.Ui,
	}

	tailErr := tailor.Run(systemID, *query)
	if tailErr != nil {
		c.Ui.Error(tailErr.Error())
		return 1
	}
	return 0
}

func (c *TailCommand) Synopsis() string {
	return "Tail logs for given private ec2 hostname"
}

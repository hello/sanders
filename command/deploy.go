package command

import (
	"fmt"
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/ec2"
	"github.com/mitchellh/cli"
	"sort"
	"strings"
)

type DeployCommand struct {
	Ui cli.ColoredUi
}

func (c *DeployCommand) Help() string {
	helpText := `
Usage: hello status $appname ...
`
	return strings.TrimSpace(helpText)
}

func (c *DeployCommand) Run(args []string) int {
	auth, _ := aws.EnvAuth()
	ecSquare := ec2.New(auth, aws.USEast)

	appName := args[0]
	c.Ui.Output(fmt.Sprintf("↳ Deploying new app for : %s", appName))

	// Find the newest ami matching app name
	filter := ec2.NewFilter()
	filter.Add("tag:suripu-app-prod", appName)

	amiResp, err := ecSquare.Images(nil, filter)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error: %s", err))
	}

	amiNames := make([]string, 0)
	amiStore := make(map[string]string)
	for _, image := range amiResp.Images {
		amiNames = append(amiNames, image.Name)
		amiStore[image.Name] = image.Id
	}

	c.Ui.Output(strings.Join(amiNames, ";"))
	sort.Strings(amiNames)
	// sort.Sort(sort.Reverse(sort.StringSlice(amiNames)))
	c.Ui.Output(strings.Join(amiNames, ";"))
	c.Ui.Info(fmt.Sprintf("Using: %s %s", amiNames[0], amiStore[amiNames[0]]))
	c.Ui.Output("")

	sgs := make([]ec2.SecurityGroup, 1)
	sgs[0] = ec2.SecurityGroup{Id: "sg-5efc4f31"}

	options := ec2.RunInstancesOptions{ImageId: amiStore[amiNames[0]], MinCount: 1, MaxCount: 1, InstanceType: "t1.micro", SubnetId: "subnet-0a524964", SecurityGroups: sgs}
	runInstancesResp, err := ecSquare.RunInstances(&options)

	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error running instance: %s", err))
	}

	c.Ui.Info(fmt.Sprintf("ReservationId: %s", runInstancesResp.RequestId))
	c.Ui.Output("Please check status of creating in the instance in the console at the moment.")

	return 0
}

func (c *DeployCommand) Synopsis() string {
	return "Tell hello to deploy a new version of the app"
}

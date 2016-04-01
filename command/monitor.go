package command

import (
  "fmt"
  "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
  "github.com/mitchellh/cli"
  "strings"
  "strconv"
  "time"
)

type MonitorCommand struct {
	Ui       cli.ColoredUi
	Notifier BasicNotifier
}

func (c *MonitorCommand) Help() string {
	helpText := `Usage: sanders monitor`
	return strings.TrimSpace(helpText)
}

func (c *MonitorCommand) Run(args []string) int {

	config := &aws.Config{
		Region: aws.String("us-east-1"),
	}

  elbs := []string{
		"suripu-service-prod",
		"suripu-app-prod",
		"suripu-app-canary",
		"suripu-admin-prod",
		"messeji-prod",
		"hello-time-prod",
	}

	service := elb.New(session.New(), config)
	ec2Service := ec2.New(session.New(), config)

  for idx, elb := range elbs {
		c.Ui.Output(fmt.Sprintf("[%d] %s", idx, elb))
	}

	elbSel, err := c.Ui.Ask("Select an elb #: ")
	elbIdx, _ := strconv.Atoi(elbSel)

	if err != nil || elbIdx >= len(elbs) {
		c.Ui.Error(fmt.Sprintf("Incorrect elb selection: %s\n", err))
		return 1
	}

	selectedElb := elbs[elbIdx]
  status := elbStatus(selectedElb, service, ec2Service)

  for {
    printStatus(c.Ui, status)
    c.Ui.Output("\nSleeping for 10 seconds...\n")
    time.Sleep(10000 * time.Millisecond)
  }

  return 0
}


func (c *MonitorCommand) Synopsis() string {
	return "Monitor ELB status"
}

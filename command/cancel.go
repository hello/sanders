package command

import (
	"fmt"
	"github.com/hello/sanders/core"
	"github.com/mitchellh/cli"
	"strings"
)

type CancelCommand struct {
	Ui           cli.ColoredUi
	Notifier     BasicNotifier
	AmiSelector  core.AmiSelector
	KeyService   core.KeyService
	Apps         []core.SuripuApp
	FleetManager *core.FleetManager
}

func (c *CancelCommand) Help() string {
	helpText := `Usage: sanders cancel-spot`
	return strings.TrimSpace(helpText)
}

func (c *CancelCommand) Run(args []string) int {
	err := c.FleetManager.Describe()
	if err != nil {
		// Message from an error.
		c.Ui.Error(fmt.Sprintf("Failed to describe Spot Fleet request: %s", err))
		return 1
	}

	requestId, err := c.Ui.Ask("Which spot request?")
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if strings.TrimSpace(requestId) == "" {
		c.Ui.Error("Empty requestId")
		return 1
	}
	c.FleetManager.Cancel(requestId)
	c.Ui.Output(fmt.Sprintf("Spot Fleet request %s was successfully cancelled", requestId))

	return 0
}

func (c *CancelCommand) Synopsis() string {
	return "Cancels a Spot Fleet request."
}

package command

import (
	"fmt"
	"github.com/crowdmob/goamz/aws"
	"github.com/mitchellh/cli"
	"strconv"
	"strings"
)

type PillCommand struct {
	Ui cli.ColoredUi
}

func (c *PillCommand) Help() string {
	helpText := `Usage: hello pill $key|$csv`
	return strings.TrimSpace(helpText)
}

func (c *PillCommand) Run(args []string) int {
	return 0
}

func (c *PillCommand) Synopsis() string {
	return "Upload pill key and csv to Hello HQ"
}

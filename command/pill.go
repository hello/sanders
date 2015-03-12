package command

import (
	"fmt"
	"github.com/crowdmob/goamz/aws"
	"github.com/mitchellh/cli"
	"strconv"
	"strings"
)

type BuildCommand struct {
	Ui cli.ColoredUi
}

func (c *BuildCommand) Help() string {
	helpText := `Usage: hello pill $key|$csv`
	return strings.TrimSpace(helpText)
}

func (c *BuildCommand) Run(args []string) int {
	return 0
}

func (c *BuildCommand) Synopsis() string {
	return "Upload pill key and csv to Hello HQ"
}

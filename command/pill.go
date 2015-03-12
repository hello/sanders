package command

import (
	"fmt"
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/s3"
	"github.com/mitchellh/cli"
	/*
	 *"strconv"
	 */
	"strings"
)

type PillCommand struct {
	Ui cli.ColoredUi
}

func (c *PillCommand) Help() string {
	helpText := `Usage: hello pill [$key|$csv]`
	return strings.TrimSpace(helpText)
}

func (c *PillCommand) Run(args []string) int {
	if 0 == len(args) {
		c.Ui.Error(fmt.Sprintf("Please provide a file name."))
	} else {
		for _, fname := range args {
			upload(c, fname)
		}
	}
	return 0
}

func (c *PillCommand) Synopsis() string {
	return "Upload pill key and csv to Hello HQ"
}

func upload(c *PillCommand, fname string) {
	c.Ui.Info(fmt.Sprintf("Uploading file %s", fname))
}

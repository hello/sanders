package command

import (
	"fmt"
	"github.com/mitchellh/cli"
)

type VersionCommand struct {
	Ui        cli.ColoredUi
	GitCommit string
}

func (c *VersionCommand) Help() string {
	return ""
}

func (c *VersionCommand) Run(args []string) int {
	c.Ui.Info(fmt.Sprintf("Version: %s\n", c.GitCommit))
	return 0
}

func (c *VersionCommand) Synopsis() string {
	return "Prints sanders's version"
}

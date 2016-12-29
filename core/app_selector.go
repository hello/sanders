package core

import (
	"errors"
	"fmt"
	"github.com/mitchellh/cli"
	"strconv"
)

type AppSelector interface {
	Choose(apps []SuripuApp) (*SuripuApp, error)
}

type CliAppSelector struct {
	Ui cli.ColoredUi
}

func NewCliAppSelector(ui cli.ColoredUi) AppSelector {
	return &CliAppSelector{
		Ui: ui,
	}
}

func (c *CliAppSelector) Choose(apps []SuripuApp) (*SuripuApp, error) {
	c.Ui.Output("Which app are we building for?")

	for idx, app := range apps {
		c.Ui.Output(fmt.Sprintf("[%d] %s", idx, app.Name))
	}

	appSel, err := c.Ui.Ask("Select an app #: ")
	appIdx, _ := strconv.Atoi(appSel)

	if err != nil {
		return nil, err
	} else if appIdx >= len(apps) {
		return nil, errors.New(fmt.Sprintf("Incorrect app selection: %s\n", err))
	}

	selectedApp := apps[appIdx]
	return &selectedApp, nil
}

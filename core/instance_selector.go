package core

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mitchellh/cli"
	"strconv"
)

type InstanceSelector interface {
	Choose(instances []*ec2.Instance) (*ec2.Instance, error)
}

type CliInstanceSelector struct {
	Ui cli.ColoredUi
}

func NewCliInstanceSelector(ui cli.ColoredUi) InstanceSelector {
	return &CliInstanceSelector{
		Ui: ui,
	}
}

func (c *CliInstanceSelector) Choose(instances []*ec2.Instance) (*ec2.Instance, error) {
	c.Ui.Output("Which app are we building for?")

	for idx, instance := range instances {
		if instance.PrivateIpAddress != nil {

			c.Ui.Output(fmt.Sprintf("[%d] %s (%s)", idx, *instance.PrivateIpAddress, *instance.KeyName))
		}
	}

	appSel, err := c.Ui.Ask("Select an instance #: ")
	appIdx, _ := strconv.Atoi(appSel)

	if err != nil {
		return nil, err
	} else if appIdx >= len(instances) {
		return nil, errors.New(fmt.Sprintf("Incorrect app selection: %s\n", err))
	}

	selected := instances[appIdx]
	return selected, nil
}

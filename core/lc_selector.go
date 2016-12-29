package core

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/mitchellh/cli"
	"sort"
	"strconv"
	"strings"
)

type LaunchConfigurationSelector interface {
	Choose(app *SuripuApp) (string, error)
}

func NewCliLaunchConfigurationSelector(ui cli.ColoredUi, asg *autoscaling.AutoScaling) *CliLaunchConfigurationSelector {
	return &CliLaunchConfigurationSelector{
		Ui:      ui,
		service: asg,
	}
}

type CliLaunchConfigurationSelector struct {
	Ui      cli.ColoredUi
	service *autoscaling.AutoScaling
}

func (c *CliLaunchConfigurationSelector) Choose(app *SuripuApp) (string, error) {
	lcParams := &autoscaling.DescribeLaunchConfigurationsInput{
		MaxRecords: aws.Int64(100),
	}

	pageNum := 0
	appPossibleLCs := make([]*autoscaling.LaunchConfiguration, 0)

	pageErr := c.service.DescribeLaunchConfigurationsPages(lcParams, func(page *autoscaling.DescribeLaunchConfigurationsOutput, lastPage bool) bool {
		pageNum++
		if len(page.LaunchConfigurations) == 0 {
			c.Ui.Error(fmt.Sprintf("No launch configuration found for app: %s", app.Name))
			return false
		}

		for _, stuff := range page.LaunchConfigurations {
			if strings.HasPrefix(*stuff.LaunchConfigurationName, app.Name) {
				appPossibleLCs = append(appPossibleLCs, stuff)
			}
		}
		return pageNum <= 2 //Allow for 200 possible LCs
	})

	if pageErr != nil {
		return "", errors.New(fmt.Sprintf("%s", pageErr))
	}

	c.Ui.Info("Latest 5 Launch Configurations")
	c.Ui.Info(" #\tName:                               \tCreated At:")
	c.Ui.Info("---|---------------------------------------|---------------")
	sort.Sort(sort.Reverse(ByLCTime(appPossibleLCs)))

	numLCs := Min(len(appPossibleLCs), 5)
	for lcIdx := 0; lcIdx < numLCs; lcIdx++ {
		c.Ui.Info(fmt.Sprintf("[%d]\t%-36s\t%s", lcIdx, *appPossibleLCs[lcIdx].LaunchConfigurationName, appPossibleLCs[lcIdx].CreatedTime.String()))
	}

	c.Ui.Output("")
	lc, err := c.Ui.Ask("Select a launch configuration (LC) #: ")
	lcNum, _ := strconv.Atoi(lc)

	if err != nil || lcNum >= len(appPossibleLCs) {
		return "", errors.New(fmt.Sprintf("Error reading app #: %s", err))
	}

	lcName := *appPossibleLCs[lcNum].LaunchConfigurationName

	return lcName, nil
}

package command

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/mitchellh/cli"
	"sort"
	"strings"
)

type CleanCommand struct {
	Ui       cli.ColoredUi
	Notifier BasicNotifier
	Services *AmznServices
}

func (c *CleanCommand) Help() string {
	helpText := `Usage: clean`
	return strings.TrimSpace(helpText)
}

func (c *CleanCommand) Run(args []string) int {

	lcParams := &autoscaling.DescribeLaunchConfigurationsInput{
		MaxRecords: aws.Int64(100),
	}

	pageNum := 0
	allLcs := make([]*autoscaling.LaunchConfiguration, 0)

	pageErr := c.Services.Asg.DescribeLaunchConfigurationsPages(lcParams, func(page *autoscaling.DescribeLaunchConfigurationsOutput, lastPage bool) bool {
		pageNum++
		if len(page.LaunchConfigurations) == 0 {
			return false
		}

		for _, lc := range page.LaunchConfigurations {
			for _, app := range suripuApps {
				if strings.HasPrefix(*lc.LaunchConfigurationName, app.name) {
					allLcs = append(allLcs, lc)
				}
			}

		}
		return pageNum <= 1 //Allow for page * max records
	})

	if pageErr != nil {
		c.Ui.Error(fmt.Sprintf("%s", pageErr))
		return 1
	}
	sort.Sort(ByLCTime(allLcs))

	lcToDelete := make([]*string, 0)
	for i, lcName := range allLcs {
		if i < 10 {
			c.Ui.Warn(*lcName.LaunchConfigurationName)
			lcToDelete = append(lcToDelete, lcName.LaunchConfigurationName)
		}
	}

	ok, err := c.Ui.Ask("Each above LCs will be deleted. Type ok to confirm.")
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%v", err))
		return 1
	}
	if ok != "ok" {
		c.Ui.Error("Didn't get ok. Bailing.")
		return 1
	}

	for _, lcName := range lcToDelete {
		params := &autoscaling.DeleteLaunchConfigurationInput{
			LaunchConfigurationName: lcName,
		}
		_, err := c.Services.Asg.DeleteLaunchConfiguration(params)

		if err != nil {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			c.Ui.Error(err.Error())

		}
	}

	return 0
}

func (c *CleanCommand) Synopsis() string {
	return "Deletes (10) oldest launch configurations that are not attached to an ASG."
}

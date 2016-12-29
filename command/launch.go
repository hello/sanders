package command

import (
	"encoding/base64"
	"fmt"
	"github.com/hello/sanders/core"
	"github.com/mitchellh/cli"
	"strings"
	"time"
)

type LaunchCommand struct {
	Ui           cli.ColoredUi
	Notifier     BasicNotifier
	AmiSelector  core.AmiSelector
	KeyService   core.KeyService
	Apps         []core.SuripuApp
	FleetManager *core.FleetManager
}

func (c *LaunchCommand) Help() string {
	helpText := `Usage: sanders launch-spot`
	return strings.TrimSpace(helpText)
}

func (c *LaunchCommand) Run(args []string) int {

	environment := "prod"

	c.Ui.Output(fmt.Sprintf("Creating LC for %s environment.\n", environment))

	appSelector := core.NewCliAppSelector(c.Ui)
	selectedApp, err := appSelector.Choose(c.Apps)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	selectedAmi, err := c.AmiSelector.Select(*selectedApp, environment)

	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Info(fmt.Sprintf("You selected %s\n", selectedAmi.Name))
	c.Ui.Info(fmt.Sprintf("Version Number: %s\n", selectedAmi.Version))

	emergencyText := "-spot"

	launchConfigName := fmt.Sprintf("%s-%s-%s%s", selectedApp.Name, environment, selectedAmi.Version, emergencyText)

	//Create deployment-specific KeyPair

	keyName := fmt.Sprintf("%s-%d", launchConfigName, time.Now().Unix())

	keyUploadResults, err := c.KeyService.Upload(keyName, *selectedApp, environment)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Info(fmt.Sprintf("Created KeyPair: %s. \n", keyUploadResults.KeyName))

	config, err := c.FleetManager.Create(selectedApp, selectedAmi, keyUploadResults.KeyName)
	if err != nil {
		c.Ui.Error(err.Error())
		c.Cleanup(keyUploadResults)
		return 1
	}

	deployAction := NewDeployAction("launch", selectedApp.Name, launchConfigName, 0)

	c.Ui.Info(fmt.Sprint("Creating Spot Fleet request with the following parameters:"))
	c.Ui.Info(fmt.Sprintf("%s", config))

	decoded, _ := base64.RawStdEncoding.DecodeString(selectedAmi.UserData)
	c.Ui.Info(fmt.Sprintf("%s", decoded))

	ok, err := c.Ui.Ask("'ok' if you agree, anything else to cancel: ")
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		c.Cleanup(keyUploadResults)
		return 1
	}

	if ok != "ok" {
		c.Ui.Warn("Cancelled.")
		if !c.Cleanup(keyUploadResults) {
			return 1
		}
		return 0
	}

	requestId, err := c.FleetManager.Execute(config)

	if err != nil {
		// Message from an error.
		c.Ui.Error(fmt.Sprintf("Failed to create Spot Fleet request: %s", err))
		c.Cleanup(keyUploadResults)
		return 1
	}

	c.Notifier.Notify(deployAction)
	c.Ui.Output(fmt.Sprintf("Spot Fleet request %s was successfully created", requestId))

	return 0
}

func (c *LaunchCommand) Cleanup(uploadRes *core.KeyUploadResult) bool {

	c.Ui.Info("")
	c.Ui.Info(fmt.Sprintf("Cleaning up created KeyPair: %s", uploadRes.KeyName))

	err := c.KeyService.CleanUp(uploadRes)
	if err != nil {
		c.Ui.Error(err.Error())
		return false
	}

	c.Ui.Info(fmt.Sprintf("Successfully deleted S3 object: %s", uploadRes.Key))

	return true
}

func (c *LaunchCommand) Synopsis() string {
	return "Launches a Spot Fleet request."
}

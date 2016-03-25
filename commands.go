package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hello/sanders/command"
	"github.com/hello/sanders/ui"
	"github.com/mitchellh/cli"
	"os"
	"os/signal"
)

// Commands is the mapping of all the available Serf commands.
var Commands map[string]cli.CommandFactory

var (
	UiColorBlack = cli.UiColor{37, false}
)

func init() {
	config := &aws.Config{
		Region: aws.String("us-east-1"),
	}
	cui := cli.ColoredUi{
		InfoColor:  cli.UiColorGreen,
		ErrorColor: cli.UiColorRed,
		WarnColor:  cli.UiColorYellow,
		Ui: &cli.BasicUi{
			Writer: os.Stdout,
			Reader: os.Stdin,
		},
	}

	iamService := iam.New(session.New(), config)
	getUserReq := &iam.GetUserInput{}

	resp, err := iamService.GetUser(getUserReq)

	if err != nil {
		cui.Ui.Error(fmt.Sprintln(err.Error()))
		return
	}

	user := *resp.User.UserName
	cpui := ui.ProgressUi{
		Writer: os.Stdout,
		Ui:     cui,
	}

	notifier := command.NewSlackNotifier(user)

	Commands = map[string]cli.CommandFactory{

		"status": func() (cli.Command, error) {
			return &command.StatusCommand{
				Ui:       cui,
				Notifier: notifier,
			}, nil
		},
		"sunset": func() (cli.Command, error) {
			return &command.SunsetCommand{
				Ui:       cui,
				Notifier: notifier,
			}, nil
		},
		"deploy": func() (cli.Command, error) {
			return &command.DeployCommand{
				Ui:       cui,
				Notifier: notifier,
			}, nil
		},
		"hosts": func() (cli.Command, error) {
			return &command.HostsCommand{
				Ui:       cui,
				Notifier: notifier,
			}, nil
		},
		"canary": func() (cli.Command, error) {
			return &command.CanaryCommand{
				Ui:       cpui,
				Notifier: notifier,
			}, nil
		},
		"confirm": func() (cli.Command, error) {
			return &command.ConfirmCommand{
				Ui:       cui,
				Notifier: notifier,
			}, nil
		},
		"create": func() (cli.Command, error) {
			return &command.CreateCommand{
				Ui:       cui,
				Notifier: notifier,
			}, nil
		},
		"version": func() (cli.Command, error) {
			return &command.VersionCommand{
				Ui:        cui,
				GitCommit: GitCommit,
			}, nil
		},
	}
}

// makeShutdownCh returns a channel that can be used for shutdown
// notifications for commands. This channel will send a message for every
// interrupt received.
func makeShutdownCh() <-chan struct{} {
	resultCh := make(chan struct{})

	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt)
	go func() {
		for {
			<-signalCh
			resultCh <- struct{}{}
		}
	}()

	return resultCh
}

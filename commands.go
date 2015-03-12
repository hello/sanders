package main

import (
	"./command"
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

	cui := cli.ColoredUi{
		InfoColor:  cli.UiColorGreen,
		ErrorColor: cli.UiColorRed,
		WarnColor:  cli.UiColorYellow,
		Ui: &cli.BasicUi{
			Writer: os.Stdout,
			Reader: os.Stdin,
		},
	}
	Commands = map[string]cli.CommandFactory{

		"status": func() (cli.Command, error) {
			return &command.StatusCommand{
				Ui: cui,
			}, nil
		},
		"deploy": func() (cli.Command, error) {
			return &command.DeployCommand{
				Ui: cui,
			}, nil
		},
		"build": func() (cli.Command, error) {
			return &command.BuildCommand{
				Ui: cui,
			}, nil
		},
		"flip": func() (cli.Command, error) {
			return &command.FlipCommand{
				Ui: cui,
			}, nil
		},
		"asg": func() (cli.Command, error) {
			return &command.ASGCommand{
				Ui: cui,
			}, nil
		},
		"hosts": func() (cli.Command, error) {
			return &command.HostsCommand{
				Ui: cui,
			}, nil
		},
		"pill": func() (cli.Command, error) {
			return &command.PillCommand{
				Ui: cui,
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

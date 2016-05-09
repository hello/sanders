package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
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
	cui := cli.ColoredUi{
		InfoColor:  cli.UiColorGreen,
		ErrorColor: cli.UiColorRed,
		WarnColor:  cli.UiColorYellow,
		Ui: &cli.BasicUi{
			Writer: os.Stdout,
			Reader: os.Stdin,
		},
	}

	config := &aws.Config{
		Region: aws.String("us-east-1"),
	}
	keyConfig := &aws.Config{
		Region: aws.String("us-west-1"),
	}

	sess := session.New()
	amzn := &command.AmznServices{
		Iam:    iam.New(sess, config),
		Asg:    autoscaling.New(sess, config),
		Ec2:    ec2.New(sess, config),
		Elb:    elb.New(sess, config),
		S3:     s3.New(sess, config),
		S3Keys: s3.New(sess, keyConfig),
	}

	getUserReq := &iam.GetUserInput{}

	resp, err := amzn.Iam.GetUser(getUserReq)

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
		"canary": func() (cli.Command, error) {
			return &command.CanaryCommand{
				Ui:       cpui,
				Notifier: notifier,
			}, nil
		},
		"clean": func() (cli.Command, error) {
			return &command.CleanCommand{
				Ui:       cui,
				Services: amzn,
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
		"monitor": func() (cli.Command, error) {
			return &command.MonitorCommand{
				Ui:       cui,
				Notifier: notifier,
			}, nil
		},
		"status": func() (cli.Command, error) {
			return &command.StatusCommand{
				Ui:       cui,
				Notifier: notifier,
				Services: amzn,
			}, nil
		},
		"sunset": func() (cli.Command, error) {
			return &command.SunsetCommand{
				Ui:       cui,
				Notifier: notifier,
				Services: amzn,
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

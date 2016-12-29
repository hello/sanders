package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hello/sanders/command"
	"github.com/hello/sanders/core"
	"github.com/mitchellh/cli"
	"os"
	"os/signal"
)

// Commands is the mapping of all the available Serf commands.
var Commands map[string]cli.CommandFactory

var (
	UiColorBlack = cli.UiColor{37, false}

	//This hash should be updated anytime default_userdata.sh is updated on S3
	expectedUserDataHash = "0011ed8a3aeaffa830620d16e39f84549cb0c6cb"
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

	sess := session.New()

	config := &aws.Config{
		Region: aws.String("us-east-1"),
	}
	s3KeyConfig := &aws.Config{
		Region: aws.String("us-west-1"),
	}

	s3service := s3.New(sess, config)
	asgService := autoscaling.New(sess, config)
	ec2service := ec2.New(sess, config)
	s3KeyService := s3.New(sess, s3KeyConfig)
	iamService := iam.New(sess, config)

	userDataGenerator := core.NewUserMetaDataGenerator(
		expectedUserDataHash,
		"hello-deploy",
		"userdata/default_userdata.sh",
		s3service,
	)

	keyService := core.NewS3KeyService(
		s3KeyService,
		ec2service,
		"hello-keys",
	)

	amiSelector := core.NewSuripuAppAmiSelector(
		cui,
		ec2service,
		s3service,
		userDataGenerator,
	)

	fleetManager := core.NewFleetManager(cui, ec2service)
	getUserReq := &iam.GetUserInput{}

	resp, err := iamService.GetUser(getUserReq)

	if err != nil {
		cui.Ui.Error(fmt.Sprintln(err.Error()))
		return
	}

	user := *resp.User.UserName
	// cpui := ui.ProgressUi{
	// 	Writer: os.Stdout,
	// 	Ui:     cui,
	// }

	notifier := command.NewSlackNotifier(user)

	Commands = map[string]cli.CommandFactory{
		"cancel-spot": func() (cli.Command, error) {
			return &command.CancelCommand{
				Ui:           cui,
				Notifier:     notifier,
				Apps:         suripuApps,
				FleetManager: fleetManager,
			}, nil
		},
		"clean": func() (cli.Command, error) {
			return &command.CleanCommand{
				Ui:   cui,
				Apps: suripuApps,
			}, nil
		},
		"confirm": func() (cli.Command, error) {
			return &command.ConfirmCommand{
				Ui:       cui,
				Notifier: notifier,
				Apps:     suripuApps,
			}, nil
		},
		"create": func() (cli.Command, error) {
			return &command.CreateCommand{
				Ui:          cui,
				Notifier:    notifier,
				AmiSelector: amiSelector,
				KeyService:  keyService,
				Ec2Service:  ec2service,
				S3Service:   s3service,
				AsgService:  asgService,
				Apps:        suripuApps,
			}, nil
		},
		"deploy": func() (cli.Command, error) {
			return &command.DeployCommand{
				Ui:       cui,
				Notifier: notifier,
				Apps:     suripuApps,
			}, nil
		},
		"hosts": func() (cli.Command, error) {
			return &command.HostsCommand{
				Ui:       cui,
				Notifier: notifier,
				Apps:     suripuApps,
			}, nil
		},
		"launch-spot": func() (cli.Command, error) {
			return &command.LaunchCommand{
				Ui:           cui,
				Notifier:     notifier,
				AmiSelector:  amiSelector,
				KeyService:   keyService,
				Apps:         suripuApps,
				FleetManager: fleetManager,
			}, nil
		},
		"monitor": func() (cli.Command, error) {
			return &command.MonitorCommand{
				Ui:       cui,
				Notifier: notifier,
				Apps:     suripuApps,
			}, nil
		},

		"setup": func() (cli.Command, error) {
			return &command.SetupCommand{
				Ui:     cui,
				Config: config,
				Apps:   suripuApps,
			}, nil
		},
		"status": func() (cli.Command, error) {
			return &command.StatusCommand{
				Ui:       cui,
				Notifier: notifier,
				Apps:     suripuApps,
			}, nil
		},
		"sunset": func() (cli.Command, error) {
			return &command.SunsetCommand{
				Ui:       cui,
				Notifier: notifier,
				Apps:     suripuApps,
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

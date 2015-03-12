package command

import (
	"fmt"
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/s3"
	"github.com/mitchellh/cli"
	/*
	 *"strconv"
	 */
	"strings"
)

type PillCommand struct {
	Ui cli.ColoredUi
}

func (c *PillCommand) Help() string {
	helpText := `Usage: hello pill [$key|$csv]`
	return strings.TrimSpace(helpText)
}

func (c *PillCommand) Run(args []string) int {
	c.Ui.Info(fmt.Sprintf("Establishing connection using AccessKey  %s", AwsAccessKey))
	auth := aws.Auth{
		AccessKey: AwsAccessKey,
		SecretKey: AwsSecretKey,
	}
	connection := s3.New(auth, aws.USEast)
	bucket := connection.Bucket("hello-firmware")
	res, err := bucket.List("", "", "", 1000)

	if err != nil {
		c.Ui.Error(fmt.Sprintf("Connection Failed %s", err))
		return 1
	} else {
		for _, v := range res.Contents {
			c.Ui.Info(fmt.Sprintf("%s", v.Key))
		}
	}

	if 0 == len(args) {
		c.Ui.Error(fmt.Sprintf("Please provide a file name."))
	} else {
		for _, fname := range args {
			upload(c, fname)
		}
	}
	return 0
}

func (c *PillCommand) Synopsis() string {
	return "Upload pill key and csv to Hello HQ"
}

func upload(c *PillCommand, fname string) {
	c.Ui.Info(fmt.Sprintf("Uploading file %s", fname))
}

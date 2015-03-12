package command

import (
	"fmt"
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/s3"
	"github.com/mitchellh/cli"
	"os"
	"path/filepath"
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
	bucket := connection.Bucket("hello-jabil")
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

func determineType(fname string) string {
	p, _ := filepath.Abs(fname)
	if _, err := os.Stat(p); err == nil {
		basename := filepath.Base(p)
		if _, pe := filepath.Match(`*.csv`, basename); pe == nil {
			return "csv"
		} else if _, pe := filepath.Match(`90500007*`, basename); pe == nil {
			if 20 <= len(basename) && len(basename) <= 21 {
				return "pill"
			}
		}
	}
	return "unkown"
}
func uploadObj(k string, v string) error {
	return nil
}
func upload(c *PillCommand, fname string) error {
	t := determineType(fname)

	if t == "pill" {
		c.Ui.Info(fmt.Sprintf("Uploading %s %s", t, fname))
	} else if t == "csv" {
		c.Ui.Info(fmt.Sprintf("Uploading %s %s", t, fname))
	} else {
		c.Ui.Warn(fmt.Sprintf("Invalid Object %s", fname))
	}
	return nil
}
func connectToS3(access string, secret string) *s3.Bucket {
	auth := aws.Auth{
		AccessKey: access,
		SecretKey: secret,
	}
	connection := s3.New(auth, aws.USEast)
	return connection.Bucket("hello-jabil")
}

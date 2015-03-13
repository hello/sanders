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

var bucket s3.Bucket

func (c *PillCommand) Help() string {
	helpText := `Usage: hello pill [$key|$csv]`
	return strings.TrimSpace(helpText)
}

func (c *PillCommand) Run(args []string) int {
	c.Ui.Info(fmt.Sprintf("Establishing connection...", AwsAccessKey))
	/*
	 *bucket := connectToS3(AwsAccessKey, AwsSecretKey)
	 */

	if 0 == len(args) {
		c.Ui.Error(fmt.Sprintf("Please provide a file name."))
	} else {
		for _, fname := range args {
			err := upload(fname)
			if err != nil {
				c.Ui.Error(fmt.Sprintf("Uploading %s failed. Error: %s.", fname, err))
				return 1
			}
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
func putObj(k string, full_name string) error {
	file, err := os.Open(full_name)
	if err != nil {
		return err
	} else {
		defer file.Close()
	}
	fileInfo, _ := file.Stat()
	var size int64 = fileInfo.Size()
	return bucket.PutReader(k, file, size, "application/octet-stream", s3.Private)
}
func upload(fname string) error {
	t := determineType(fname)
	if t == "unknown" {
		return fmt.Errorf("Invalid Object %s", fname)
	} else {
		full_name, _ := filepath.Abs(fname)
		key := t + filepath.Base(full_name)
		return putObj(key, full_name)
	}
}
func connectToS3(access string, secret string) *s3.Bucket {
	auth := aws.Auth{
		AccessKey: access,
		SecretKey: secret,
	}
	connection := s3.New(auth, aws.USEast)
	return connection.Bucket("hello-jabil")
}

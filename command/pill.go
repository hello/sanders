package command

import (
	"fmt"
	"github.com/mitchellh/cli"
	"gopkg.in/amz.v1/aws"
	"gopkg.in/amz.v1/s3"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type PillCommand struct {
	Ui cli.ColoredUi
}

func (c *PillCommand) Help() string {
	helpText := `Usage: hello pill [$key|$csv]`
	return strings.TrimSpace(helpText)
}

func (c *PillCommand) Run(args []string) int {
	c.Ui.Info(fmt.Sprintf("Establishing connection..."))
	bucket := connectToS3(AwsAccessKey, AwsSecretKey)

	if 0 == len(args) {
		c.Ui.Error(fmt.Sprintf("Please provide a file name."))
	} else {
		for _, fname := range args {
			err := upload(bucket, fname)
			if err != nil {
				c.Ui.Error(fmt.Sprintf("Uploading %s failed. Error: %s.", fname, err))
				return 1
			} else {
				c.Ui.Info(fmt.Sprintf("Uploaded %s", fname))
			}
		}
	}
	c.Ui.Info(fmt.Sprintf("Pass"))
	return 0
}

func (c *PillCommand) Synopsis() string {
	return "Upload pill key and csv to Hello HQ"
}

func determineKeyType(fname string) string {
	p, _ := filepath.Abs(fname)
	if _, err := os.Stat(p); err == nil {
		basename := filepath.Base(p)
		if isCSV, pe := filepath.Match(`*.csv`, basename); pe == nil && isCSV {
			return "csv"
		} else if isPill, pe := filepath.Match(`90500007*`, basename); pe == nil && isPill {
			if 20 <= len(basename) && len(basename) <= 21 {
				return "pill"
			}
		} else if isZip, pe := filepath.Match(`*.zip`, basename); pe == nil && isZip {
			return "zip"
		}
	}
	return "unkown"
}
func putObj(bucket *s3.Bucket, k string, full_name string) error {
	file, err := os.Open(full_name)
	if err != nil {
		return err
	} else {
		defer file.Close()
	}
	fileInfo, _ := file.Stat()
	return bucket.PutReader(k, file, fileInfo.Size(), "application/octet-stream", s3.Private)
}
func upload(bucket *s3.Bucket, fname string) error {
	t := determineKeyType(fname)
	if t == "unknown" {
		return fmt.Errorf("Invalid Object %s", fname)
	} else {
		full_name, _ := filepath.Abs(fname)
		key := t + `/` + time.Now().UTC().Format("20060102150405") + "-" + filepath.Base(full_name)
		return putObj(bucket, key, full_name)
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

package command

import (
	"errors"
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
	helpText := `Usage: hello pill [$zip|$key|$csv]`
	return strings.TrimSpace(helpText)
}

func (c *PillCommand) Run(args []string) int {
	c.Ui.Info(fmt.Sprintf("Establishing connection with %s", string(AwsSecretKey)[:2]))
	bucket := connectToS3(AwsAccessKey, AwsSecretKey)

	if 0 == len(args) {
		c.Ui.Error(fmt.Sprintf("Please provide a file name."))
		c.Ui.Error(fmt.Sprintf("Fail"))
		return 1
	} else {
		for _, fname := range args {
			if err := upload(bucket, fname); err == nil {
				c.Ui.Info(fmt.Sprintf("Uploaded %s", fname))
			} else {
				c.Ui.Error(fmt.Sprintf("Uploading %s failed. Error: %s.", fname, err))
				c.Ui.Error(fmt.Sprintf("Fail"))
				return 1
			}
		}
	}
	c.Ui.Info(fmt.Sprintf("Pass"))
	return 0
}

func (c *PillCommand) Synopsis() string {
	return "Upload pill key and csv to Hello HQ"
}

func determineKeyType(fname string) (string, error) {
	p, _ := filepath.Abs(fname)
	if _, err := os.Stat(p); err == nil {
		basename := filepath.Base(p)
		if isCSV, pe := filepath.Match(`*.csv`, basename); pe == nil && isCSV {
			return "csv", nil
		} else if isPill, pe := filepath.Match(`90500007*`, basename); pe == nil && isPill {
			if 20 <= len(basename) && len(basename) <= 21 {
				return "pill", nil
			}
		} else if isZip, pe := filepath.Match(`*.zip`, basename); pe == nil && isZip {
			return "zip", nil
		}
	}
	return "unknown", errors.New("Invalid Object")
}
func putObj(bucket *s3.Bucket, k string, fullName string) error {
	if file, err := os.Open(fullName); err == nil {
		defer file.Close()
		fileInfo, _ := file.Stat()
		return bucket.PutReader(k, file, fileInfo.Size(), "application/octet-stream", s3.Private)
	} else {
		return err
	}
}
func upload(bucket *s3.Bucket, fname string) error {
	if t, err := determineKeyType(fname); err == nil {
		fullName, _ := filepath.Abs(fname)
		key := t + `/` + time.Now().UTC().Format("20060102150405") + "-" + filepath.Base(fullName)
		return putObj(bucket, key, fullName)
	} else {
		return err
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

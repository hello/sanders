package command

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type PillCommand struct {
	Ui cli.ColoredUi
}

func (c *PillCommand) Help() string {
	helpText := `Usage: hello upload {Morpheus,Pill,Top}*.zip`
	return strings.TrimSpace(helpText)
}

func (c *PillCommand) Run(args []string) int {
	c.Ui.Info(fmt.Sprintf("Establishing connection with key %s", string(AwsSecretKey)[:2]))
	bucket := connectToS3(AwsAccessKey, AwsSecretKey)

	if 0 == len(args) {
		c.Ui.Error(fmt.Sprintf("Please provide a file name."))
		c.Ui.Error(fmt.Sprintf("Fail"))
		return 1
	} else {
		for _, fname := range args {
			if err := uploadAndVerify(bucket, fname); err == nil {
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
		if isPill, pe := filepath.Match(`[pP]ill_*.zip`, basename); pe == nil && isPill {
			return "zip", nil
		} else if isMorpheus, pe := filepath.Match(`[mM]orpheus_*.zip`, basename); pe == nil && isMorpheus {
			return "morpheus", nil
		} else if isTop, pe := filepath.Match(`[tT]op_*.zip`, basename); pe == nil && isTop {
			return "top", nil
		}
	}
	return "unknown", errors.New("Invalid Object")
}
func putObj(bucket *s3.Bucket, key string, content []byte) ([md5.Size]byte, error) {
	md5Sum := md5.Sum(content)
	md5B64 := base64.StdEncoding.EncodeToString(md5Sum[:])
	fmt.Printf("MD5 is %s\n", md5B64)
	headers := map[string][]string{
		"Content-Type": {"application/octet-stream"},
		"Content-MD5":  {md5B64},
	}
	return md5Sum, bucket.PutHeader(key, content, headers, s3.Private)
}
func verify(bucket *s3.Bucket, key string, md5Sum [md5.Size]byte) error {
	if content, err := bucket.Get(key); err == nil {
		gavinNewSum := md5.Sum(content)
		if !bytes.Equal(md5Sum[:], gavinNewSum[:]) {
			return errors.New(fmt.Sprintf("Mismatched MD5Sum: %s", base64.StdEncoding.EncodeToString(gavinNewSum[:])))
		}
	} else {
		return err
	}
	return nil
}
func uploadAndVerify(bucket *s3.Bucket, fname string) error {
	fullName, _ := filepath.Abs(fname)
	t, err := determineKeyType(fullName)
	if err != nil {
		return err
	}
	key := t + `/` + time.Now().UTC().Format("20060102150405") + "-" + filepath.Base(fullName)
	fileContent, err := ioutil.ReadFile(fullName)
	if err != nil {
		return err
	} else if len(fileContent) == 0 {
		return errors.New("File content can not be 0")
	}
	if md5Sum, err := putObj(bucket, key, fileContent); err == nil {
		return verify(bucket, key, md5Sum)
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

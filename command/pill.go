package command

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
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
			if err := uploadAndVerify(bucket, fname, 3, 10); err == nil {
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
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return "unknown", errors.New("File doesn't exist")
	} else if err == nil {
		basename := filepath.Base(p)
		if isPill, pe := filepath.Match(`[pP]ill_*.zip`, basename); pe == nil && isPill {
			return "zip", nil
		} else if isMorpheus, pe := filepath.Match(`[mM]orpheus_*.zip`, basename); pe == nil && isMorpheus {
			return "morpheus", nil
		} else if isTop, pe := filepath.Match(`[tT]op_*.zip`, basename); pe == nil && isTop {
			return "top", nil
		} else {
			return "unknown", errors.New("Unknown file type")
		}
	}
	return "unknown", errors.New("Invalid Object")
}
func putObj(bucket *s3.Bucket, key string, content []byte, md5B64 string) error {
	headers := map[string][]string{
		"Content-Type": {"application/octet-stream"},
		"Content-MD5":  {md5B64},
	}
	return bucket.PutHeader(key, content, headers, s3.Private)
}
func calcMD5(content []byte) ([md5.Size]byte, string) {
	md5Sum := md5.Sum(content)
	md5B64 := base64.StdEncoding.EncodeToString(md5Sum[:])
	fmt.Printf("MD5 is %s\n", md5B64)
	return md5Sum, md5B64
}
func verify(bucket *s3.Bucket, key string, md5Sum [md5.Size]byte) error {
	if content, err := bucket.Get(key); err == nil {
		gavinNewSum := md5.Sum(content)
		fmt.Printf("Uploaded MD5 is %s\n", base64.StdEncoding.EncodeToString(gavinNewSum[:]))
		if !bytes.Equal(md5Sum[:], gavinNewSum[:]) {
			return errors.New(fmt.Sprintf("Mismatched MD5Sum: %s", base64.StdEncoding.EncodeToString(gavinNewSum[:])))
		}
	} else {
		return err
	}
	return nil
}
func quickVerify(bucket *s3.Bucket, key string, md5Sum string) error {
	if content, err := bucket.GetKey(key); err == nil {
		fmt.Printf("Uploaded MD5 is %s\n", content.ETag)
		if content.ETag != md5Sum {
			return errors.New(fmt.Sprintf("Mismatched MD5Sum: %s", md5Sum))
		} else {
			return nil
		}
	} else {
		return err
	}
}
func uploadAndVerify(bucket *s3.Bucket, fname string, retryMax int, sleepSecs int) error {
	fullName, _ := filepath.Abs(fname)
	t, err := determineKeyType(fullName)
	if err != nil {
		return err
	}
	extension := filepath.Ext(fullName)
	fileName := filepath.Base(fullName)
	baseName := strings.TrimSuffix(fileName, extension)
	key := t + `/` + baseName + "-" + time.Now().UTC().Format("20060102150405") + extension
	fileContent, err := ioutil.ReadFile(fullName)
	if err != nil {
		return err
	} else if len(fileContent) == 0 {
		return errors.New("File content can not be 0")
	}
	md5Sum, md5B64 := calcMD5(fileContent)
	compStr := `"` + hex.EncodeToString(md5Sum[:]) + `"`
	listResp, err := bucket.List(t+`/`+baseName, `/`, "", 1000)
	if err != nil {
		return err
	}
	for _, key := range listResp.Contents {
		if compStr == key.ETag {
			fmt.Printf("File already exists as: %s\n", key.Key)
			return nil
		}
	}
	for retries := 0; retries < retryMax; retries++ {
		var res error
		if err := putObj(bucket, key, fileContent, md5B64); err == nil {
			if err := quickVerify(bucket, key, compStr); err == nil {
				return nil
			} else {
				res = err
			}
		} else {
			res = err
		}
		fmt.Printf("Retry: %d, %s\n", retries+1, res)
		if retries+1 != retryMax {
			time.Sleep(time.Duration(sleepSecs) * time.Second)
		}
	}
	return errors.New("Unable to Upload")
}
func connectToS3(access string, secret string) *s3.Bucket {
	auth := aws.Auth{
		AccessKey: access,
		SecretKey: secret,
	}
	connection := s3.New(auth, aws.USEast)
	return connection.Bucket("hello-jabil")
}

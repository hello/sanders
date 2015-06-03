package command

import (
	"errors"
	"fmt"
	"github.com/mitchellh/cli"
	"strconv"
	"strings"
)

type ListS3Command struct {
	Ui cli.ColoredUi
}

func (c *ListS3Command) Help() string {
	helpText := `Usage: sanders list pill [n]`
	return strings.TrimSpace(helpText)
}

func (c *ListS3Command) Run(args []string) int {
	c.Ui.Info(fmt.Sprintf("Establishing connection with key %s", string(AwsSecretKey)[:2]))
	bucket := connectToS3(AwsAccessKey, AwsSecretKey)
	if 0 == len(args) {
		c.Ui.Error(fmt.Sprintf("Please provide a valid bin type (pill)"))
		c.Ui.Error(fmt.Sprintf("Fail"))
		return 1
	} else if ktype, err := parseKeyType(args[0]); err != nil {
		c.Ui.Error(fmt.Sprintf("Invalid Key: %s", ktype))
		c.Ui.Error(fmt.Sprintf("Fail"))
		return 1
	} else {
		listCount := 3
		if len(args) >= 2 {
			if count, err := strconv.Atoi(args[1]); err == nil {
				listCount = count
			}
		}
		c.Ui.Info(fmt.Sprintf("Viewing %s with history count = %d", ktype, listCount))
		if resp, err := bucket.List(ktype, "/", "", 1000); err == nil {
			start := 0
			contentSize := len(resp.Contents)
			if contentSize >= listCount {
				start = contentSize - listCount
			}
			for _, k := range resp.Contents[start:] {
				strs := strings.SplitAfter(k.Key, "-")
				if len(strs) >= 2 {
					c.Ui.Info(fmt.Sprintf("%s", strs[1]))
				}
			}
		} else {
			c.Ui.Error(fmt.Sprintf("List error %s", err))
			c.Ui.Error(fmt.Sprintf("Fail"))
			return 1
		}

	}
	c.Ui.Info(fmt.Sprintf("Pass"))
	return 0
}

func (c *ListS3Command) Synopsis() string {
	return "List the blobs uploaded to Hello"
}

func parseKeyType(fname string) (string, error) {
	if strings.EqualFold("pill", fname) {
		return "zip/", nil
	} else {
		return "unknown", errors.New("Invalid Object")
	}
}

package command

import (
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/mitchellh/cli"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	/*
	 *"io/ioutil"
	 */)

type CSVCommand struct {
	Ui    cli.ColoredUi
	Debug bool
}

func (c *CSVCommand) Synopsis() string {
	return "Verifies blobs exist given a csv and a folder"
}
func (c *CSVCommand) Help() string {
	helpText := `Usage: sanders csv [csv_folder] [key_folder]`
	return strings.TrimSpace(helpText)
}
func extractFolders(args []string) (string, string, error) {
	csvFolder, _ := filepath.Abs(".")
	keyFolder, _ := filepath.Abs(".")
	var err error = nil
	if len(args) < 1 {
		err = errors.New("Need to supply at least the key folder")
		goto exit
	} else if len(args) == 1 {
		keyFolder, err = filepath.Abs(args[0])
	} else {
		csvFolder, err = filepath.Abs(args[0])
		if err != nil {
			goto exit
		}
		keyFolder, err = filepath.Abs(args[1])
	}
exit:
	return csvFolder, keyFolder, err
}
func listFile(absName string, pattern string) string {
	ret, _ := filepath.Glob(filepath.Join(absName, pattern))
	if len(ret) > 0 {
		return ret[0]
	} else {
		return ""
	}
}
func listFiles(absName string, pattern string) []string {
	ret, _ := filepath.Glob(filepath.Join(absName, pattern))
	return ret
}
func (c *CSVCommand) extractRowKeys(absName string) []string {
	ret := make([]string, 0)
	if f, err := os.Open(absName); err == nil {
		defer f.Close()
		reader := csv.NewReader(f)
		records, _ := reader.ReadAll()
		for _, record := range records {
			if len(record) >= 3 {
				match, _ := regexp.MatchString("90500007*", record[2])
				if match {
					ret = append(ret, record[2])
				}
			}
		}
	} else {
		c.Ui.Warn(fmt.Sprintf("%s", err))
	}
	return ret
}
func (c *CSVCommand) Run(args []string) int {
	csvFolder, keyFolder, err := extractFolders(args)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		return -1
	}
	csvList := listFiles(csvFolder, "*.csv")
	csvKeys := make(map[string]string)
	for _, f := range csvList {
		for _, key := range c.extractRowKeys(f) {
			if _, ok := csvKeys[key]; !ok {
				csvKeys[key] = listFile(keyFolder, key)
			}
		}
	}
	if c.Debug {
		c.Ui.Info(fmt.Sprintf("Using csv root %s:", csvFolder))
		for _, f := range csvList {
			c.Ui.Info(fmt.Sprintf("\t * %s", f))
		}
		c.Ui.Info(fmt.Sprintf("Using key root %s", keyFolder))
		c.Ui.Info(fmt.Sprintf("Matching keys"))
		for k, v := range csvKeys {
			c.Ui.Info(fmt.Sprintf("\t + %s \t: %s", k, v))
		}
	}

	return 0

}

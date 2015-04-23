package command

import (
	"errors"
	"fmt"
	"github.com/mitchellh/cli"
	"path/filepath"
	"strings"
	/*
	 *"io/ioutil"
	 *"os"
	 */)

type CSVCommand struct {
	Ui cli.ColoredUi
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
func (c *CSVCommand) Run(args []string) int {
	csvFolder, keyFolder, err := extractFolders(args)
	if err != nil {
		fmt.Printf("%s", err)
		return -1
	}
	fmt.Printf("csv %s, key %s", csvFolder, keyFolder)
	return 0

}

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/mitchellh/cli"
)

// GitCommit holds the sha of the commit to identify the build version
var GitCommit string

func main() {
	os.Exit(realMain())
}

func realMain() int {
	// log.SetOutput(ioutil.Discard)

	// Get the command line args. We shortcut "--version" and "-v" to
	// just show the version.
	args := os.Args[1:]
	debug := false
	for _, arg := range args {
		if arg == "-v" || arg == "--version" {
			newArgs := make([]string, len(args)+1)
			newArgs[0] = "version"
			copy(newArgs[1:], args)
			args = newArgs
			break
		} else if arg == "-d" {
			debug = true
		}
	}

	if !debug {
		log.SetOutput(ioutil.Discard)
	}

	cli := &cli.CLI{
		Name:     "sanders",
		Args:     args,
		Commands: Commands,
	}

	exitCode, err := cli.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing CLI: %s\n", err.Error())
		return 1
	}

	return exitCode
}

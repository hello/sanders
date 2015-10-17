package ui

import (
	"fmt"
	"github.com/mitchellh/cli"
	"io"
)

type ProgressUi struct {
	Writer io.Writer
	Ui     cli.ColoredUi
}

func (u *ProgressUi) Progress(message string) {
	fmt.Fprint(u.Writer, message)
	fmt.Fprint(u.Writer, "\r")
}

func (u *ProgressUi) Ask(message string) (string, error) {
	return u.Ui.Ask(message)
}

func (u *ProgressUi) AskSecret(message string) (string, error) {
	return u.Ui.AskSecret(message)
}

func (u *ProgressUi) Output(message string) {
	u.Ui.Output(message)
}

func (u *ProgressUi) Info(message string) {
	u.Ui.Info(message)
}

func (u *ProgressUi) Error(message string) {
	u.Ui.Error(message)
}

func (u *ProgressUi) Warn(message string) {
	u.Ui.Warn(message)
}

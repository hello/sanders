package setup

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/multistep"
)

type StepLaunchConfiguration struct {
	ImageId        string
	SecurityGroups []string
	KeyName        string
	AppName        string
	InstanceType   string
}

func (s *StepLaunchConfiguration) Run(state multistep.StateBag) multistep.StepAction {

	srv := state.Get("asg").(*autoscaling.AutoScaling)

	ui := state.Get("ui").(cli.ColoredUi)

	lcVersionedName := fmt.Sprintf("%s-0.0.0", s.AppName)
	state.Put("lc_name", lcVersionedName)
	createLcInput := &autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName: aws.String(lcVersionedName),
		ImageId:                 aws.String(s.ImageId),
		SecurityGroups:          aws.StringSlice(s.SecurityGroups),
		KeyName:                 aws.String(s.KeyName),
		InstanceType:            aws.String(s.InstanceType),
	}

	_, err := srv.CreateLaunchConfiguration(createLcInput)
	if err != nil {
		ui.Error(fmt.Sprintf("%s", err))
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (s *StepLaunchConfiguration) Cleanup(state multistep.StateBag) {
	_, cancelled := state.GetOk(multistep.StateCancelled)
	_, halted := state.GetOk(multistep.StateHalted)
	if !cancelled && !halted {
		return
	}

	ui := state.Get("ui").(cli.ColoredUi)
	ui.Output("Cleaning up StepLaunchConfiguration")
	srv := state.Get("asg").(*autoscaling.AutoScaling)
	lcName := state.Get("lc_name").(string)
	_, err := srv.DeleteLaunchConfiguration(&autoscaling.DeleteLaunchConfigurationInput{
		LaunchConfigurationName: aws.String(lcName),
	})
	if err != nil {
		ui.Error(err.Error())
		return
	}
	ui.Info("Successfully delete launch config")
}

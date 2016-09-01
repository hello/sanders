package setup

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/multistep"
	"strings"
)

type StepCreateAutoScalingGroups struct {
	AppName string
	Subnets []string
	Azs     []string
}

func (s *StepCreateAutoScalingGroups) Run(state multistep.StateBag) multistep.StepAction {

	srv := state.Get("asg").(*autoscaling.AutoScaling)
	lcName := state.Get("lc_name").(string)

	ui := state.Get("ui").(cli.ColoredUi)

	blue := fmt.Sprintf("%s-prod", s.AppName)
	green := fmt.Sprintf("%s-prod-green", s.AppName)
	state.Put("asg_blue", blue)
	state.Put("asg_green", green)

	asgNames := []string{blue, green}
	elbName := state.Get("elb_name").(string)
	for _, asgName := range asgNames {
		createAsgInput := &autoscaling.CreateAutoScalingGroupInput{
			AutoScalingGroupName:    aws.String(asgName),
			LaunchConfigurationName: aws.String(lcName),
			VPCZoneIdentifier:       aws.String(strings.Join(s.Subnets, ",")), // so jank
			AvailabilityZones:       aws.StringSlice(s.Azs),
			DesiredCapacity:         aws.Int64(0),
			MaxSize:                 aws.Int64(0),
			MinSize:                 aws.Int64(0),
			LoadBalancerNames:       aws.StringSlice([]string{elbName}),
		}

		_, err := srv.CreateAutoScalingGroup(createAsgInput)
		if err != nil {
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}
	return multistep.ActionContinue
}

func (s *StepCreateAutoScalingGroups) Cleanup(state multistep.StateBag) {
	_, cancelled := state.GetOk(multistep.StateCancelled)
	_, halted := state.GetOk(multistep.StateHalted)
	if !cancelled && !halted {
		return
	}

	ui := state.Get("ui").(cli.ColoredUi)
	blue := state.Get("asg_blue").(string)
	green := state.Get("asg_green").(string)
	srv := state.Get("asg").(*autoscaling.AutoScaling)
	asgs := []string{blue, green}
	for _, asg := range asgs {
		_, err := srv.DeleteAutoScalingGroup(&autoscaling.DeleteAutoScalingGroupInput{
			AutoScalingGroupName: aws.String(asg),
		})
		if err != nil {
			ui.Error(err.Error())
		}
	}
	ui.Output("Cleaning up AsgStep")
}

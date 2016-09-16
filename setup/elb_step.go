package setup

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/multistep"
)

type StepCreateELB struct {
	AppName    string
	ElbOutPort int64
	ElbInPort  int64
	Subnets    []string
}

func (s *StepCreateELB) Run(state multistep.StateBag) multistep.StepAction {

	ui := state.Get("ui").(cli.ColoredUi)
	srv := state.Get("elb").(*elb.ELB)

	elbName := fmt.Sprintf("%s-prod", s.AppName)

	elbSg := state.Get("elb_sg").(string)

	input := &elb.CreateLoadBalancerInput{
		LoadBalancerName: aws.String(elbName),
		Subnets:          aws.StringSlice(s.Subnets),
		Tags: []*elb.Tag{{
			Key:   aws.String("Name"),
			Value: aws.String(elbName),
		}},
		Listeners: []*elb.Listener{
			{
				InstancePort:     aws.Int64(s.ElbOutPort),
				InstanceProtocol: aws.String("tcp"),
				LoadBalancerPort: aws.Int64(s.ElbInPort),
				Protocol:         aws.String("tcp"),
			},
		},
		Scheme:         aws.String("internet-facing"),
		SecurityGroups: aws.StringSlice([]string{elbSg}),
	}

	elbOut, err := srv.CreateLoadBalancer(input)
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	state.Put("elb_name", elbName)
	ui.Info(fmt.Sprintf("ELB %s[%s] created", elbName, *elbOut.DNSName))
	ui.Info("Don't forget to add a SSL cert")

	return multistep.ActionContinue
}

func (s *StepCreateELB) Cleanup(state multistep.StateBag) {
}

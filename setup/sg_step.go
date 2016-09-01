package setup

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/multistep"
)

type StepCreateSecurityGroups struct {
	AppName   string
	VpcId     string
	AppInPort int64
}

func (s *StepCreateSecurityGroups) Run(state multistep.StateBag) multistep.StepAction {

	ui := state.Get("ui").(cli.ColoredUi)

	srv := state.Get("ec2").(*ec2.EC2)

	elbSgName := fmt.Sprintf("elb-%s-prod", s.AppName)

	input := &ec2.CreateSecurityGroupInput{
		VpcId:       aws.String(s.VpcId),
		GroupName:   aws.String(elbSgName),
		Description: aws.String("ELB security group for " + s.AppName),
	}

	elbSgOut, err := srv.CreateSecurityGroup(input)

	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	state.Put("elb_sg", *elbSgOut.GroupId)
	ui.Info(fmt.Sprintf("%s[%s] created", elbSgName, *elbSgOut.GroupId))

	tags := &ec2.CreateTagsInput{
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(elbSgName),
			},
		},
		Resources: aws.StringSlice([]string{*elbSgOut.GroupId}),
	}

	_, err = srv.CreateTags(tags)

	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	authIngress := &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: elbSgOut.GroupId,
		IpPermissions: []*ec2.IpPermission{
			{
				FromPort:   aws.Int64(443),
				ToPort:     aws.Int64(443),
				IpProtocol: aws.String("TCP"),

				IpRanges: []*ec2.IpRange{
					{
						CidrIp: aws.String("0.0.0.0/0"),
					},
				},
			},
		},
	}

	_, err = srv.AuthorizeSecurityGroupIngress(authIngress)

	if err != nil {
		ui.Error(fmt.Sprintf("%s", err))
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Info(fmt.Sprintf("%s[%s] ingress rule created", elbSgName, *elbSgOut.GroupId))
	// APP SG

	appSgName := fmt.Sprintf("%s-prod", s.AppName)

	input = &ec2.CreateSecurityGroupInput{
		VpcId:       aws.String(s.VpcId),
		GroupName:   aws.String(appSgName),
		Description: aws.String("Security group for " + s.AppName),
	}

	out, err := srv.CreateSecurityGroup(input)

	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Info(fmt.Sprintf("%s[%s] created", appSgName, *out.GroupId))

	tags = &ec2.CreateTagsInput{
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(appSgName),
			},
		},
		Resources: aws.StringSlice([]string{*out.GroupId}),
	}

	_, err = srv.CreateTags(tags)

	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	authIngress = &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: out.GroupId,
		IpPermissions: []*ec2.IpPermission{
			{
				FromPort:   aws.Int64(s.AppInPort),
				ToPort:     aws.Int64(s.AppInPort),
				IpProtocol: aws.String("TCP"),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						GroupId: elbSgOut.GroupId,
						UserId:  aws.String("053216739513"),
					},
				},
			},
		},
	}

	_, err = srv.AuthorizeSecurityGroupIngress(authIngress)

	if err != nil {
		ui.Error(fmt.Sprintf("%s", err))
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Info(fmt.Sprintf("%s[%s] ingress rule created", appSgName, *out.GroupId))

	return multistep.ActionContinue
}

func (s *StepCreateSecurityGroups) Cleanup(state multistep.StateBag) {
}

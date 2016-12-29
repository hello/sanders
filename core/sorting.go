package core

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
)

type ByImageTime []*ec2.Image

func (s ByImageTime) Len() int {
	return len(s)
}
func (s ByImageTime) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByImageTime) Less(i, j int) bool {
	return *s[i].CreationDate < *s[j].CreationDate
}

type ByObjectLastModified []*s3.Object

func (s ByObjectLastModified) Len() int {
	return len(s)
}

func (s ByObjectLastModified) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s ByObjectLastModified) Less(i, j int) bool {
	return s[i].LastModified.Unix() < s[j].LastModified.Unix()
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

type ByLCTime []*autoscaling.LaunchConfiguration

func (s ByLCTime) Len() int {
	return len(s)
}
func (s ByLCTime) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByLCTime) Less(i, j int) bool {
	return s[i].CreatedTime.Unix() < s[j].CreatedTime.Unix()
}

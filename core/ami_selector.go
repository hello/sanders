package core

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	// "github.com/aws/aws-sdk-go/aws/session"
	// "github.com/aws/aws-sdk-go/service/autoscaling"
	"errors"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/mitchellh/cli"
	"sort"
	"strconv"
	"strings"
	"time"
)

type AmiSelector interface {
	Select(app SuripuApp, environment string) (*SelectedAmi, error)
}

type SuripuAppAmiSelector struct {
	packer *PackerAmiSelector
	lc     *LcAmiSelector
}

func (a *SuripuAppAmiSelector) Select(app SuripuApp, environment string) (*SelectedAmi, error) {
	if app.UsesPacker {
		return a.packer.Select(app, environment)
	}

	return a.lc.Select(app, environment)
}

func NewSuripuAppAmiSelector(ui cli.ColoredUi, ec2service *ec2.EC2, s3service *s3.S3, userDataGenerator *UserMetaDataGenerator) *SuripuAppAmiSelector {
	return &SuripuAppAmiSelector{
		packer: &PackerAmiSelector{
			Ui:         ui,
			ec2Service: ec2service,
		},
		lc: &LcAmiSelector{
			Ui:                ui,
			ec2Service:        ec2service,
			s3Service:         s3service,
			userdataGenerator: userDataGenerator,
		},
	}
}

type LcAmiSelector struct {
	Ui                cli.ColoredUi
	ec2Service        *ec2.EC2
	s3Service         *s3.S3
	userdataGenerator *UserMetaDataGenerator
}

type PackerAmiSelector struct {
	Ui         cli.ColoredUi
	ec2Service *ec2.EC2
}

func (a *LcAmiSelector) Select(app SuripuApp, environment string) (*SelectedAmi, error) {
	canaryPath := ""
	if environment == "canary" {
		canaryPath = "canary/"
	}
	pkgPrefix := fmt.Sprintf("packages/%s/%s/%s", app.PackagePath, app.Name, canaryPath)

	a.Ui.Info(pkgPrefix)
	a.Ui.Output("")
	//retrieve package list from S3 for selectedApp
	s3ListParams := &s3.ListObjectsV2Input{
		Bucket: aws.String("hello-deploy"), // Required
		//Delimiter:    aws.String("/"),
		//EncodingType: aws.String("EncodingType"),
		//Marker:       aws.String("Marker"),
		MaxKeys: aws.Int64(100000),
		Prefix:  aws.String(pkgPrefix),
	}

	availablePackages := make([]*s3.Object, 0)

	err := a.s3Service.ListObjectsV2Pages(s3ListParams,
		func(page *s3.ListObjectsV2Output, lastPage bool) bool {
			for _, item := range page.Contents {
				if strings.HasSuffix(*item.Key, ".deb") {
					availablePackages = append(availablePackages, item)
				}
			}
			return !lastPage
		})

	if err != nil {
		return nil, err
	}

	sort.Sort(sort.Reverse(ByObjectLastModified(availablePackages)))

	versions := make([]string, 0)

	a.Ui.Info(fmt.Sprintf("Latest 10 packages available for %s:", app.Name))
	a.Ui.Info(" #\tVersion:  \tLast Modified:")
	a.Ui.Info("---|----------------|---------------")
	numImages := Min(len(availablePackages), 10)
	for idx := 0; idx < numImages; idx++ {
		objectKeyChunks := strings.Split(*availablePackages[idx].Key, "/")
		versionNumber := objectKeyChunks[len(objectKeyChunks)-2]
		versions = append(versions, versionNumber)
		a.Ui.Output(fmt.Sprintf("[%d]\t%s\t\t%s", idx, versionNumber, availablePackages[idx].LastModified.Format(time.UnixDate)))
	}

	ver, err := a.Ui.Ask("Select a version #: ")
	verIdx, _ := strconv.Atoi(ver)

	if err != nil {
		return nil, err
	} else if verIdx >= len(availablePackages) {
		return nil, errors.New(fmt.Sprintf("Incorrect AMI selection: %s\n", err))
	}

	amiVersion := versions[verIdx]

	//Get the userdata template from S3 for instance startup using cloud-init
	metadataInput := UserMetaDataInput{
		AmiVersion:    amiVersion,
		AppName:       app.Name,
		PackagePath:   app.PackagePath,
		CanaryPath:    canaryPath,
		DefaultRegion: "us-east-1",
		JavaVersion:   app.JavaVersion,
	}

	userData, err := a.userdataGenerator.Parse(&metadataInput)
	if err != nil {
		return nil, err
	}

	amiName := "a cloud-init deploy based on the AMI: Base-2016-12-02"
	amiId := "ami-16d5ee01"

	selectedAmi := SelectedAmi{
		Id:       amiId,
		Name:     amiName,
		Version:  amiVersion,
		UserData: userData,
	}
	return &selectedAmi, nil
}

func (a *PackerAmiSelector) Select(app SuripuApp, environment string) (*SelectedAmi, error) {
	a.Ui.Warn(fmt.Sprintf("%s not yet handled by Packer-free deployment. Proceeding with Packer-created AMI selection.", app.Name))

	ec2ParamsAll := &ec2.DescribeImagesInput{
		DryRun: aws.Bool(false),
		Filters: []*ec2.Filter{
			{ // Required
				Name: aws.String("is-public"),
				Values: []*string{
					aws.String("false"), // Required
					// More values...
				},
			},
		},
	}
	respAll, err := a.ec2Service.DescribeImages(ec2ParamsAll)

	if err != nil {
		return nil, err
	}

	validImages := make([]*ec2.Image, 0)

	for _, image := range respAll.Images {
		if strings.HasPrefix(*image.Name, app.Name) {
			validImages = append(validImages, image)
		}
	}

	sort.Sort(sort.Reverse(ByImageTime(validImages)))

	a.Ui.Output("Which AMI should be used?")
	numImages := Min(len(validImages), 10)
	for idx := 0; idx < numImages; idx++ {
		a.Ui.Output(fmt.Sprintf("[%d] \t%s\t%s", idx, *validImages[idx].Name, *validImages[idx].CreationDate))
	}

	ami, err := a.Ui.Ask("Select an AMI #: ")
	amiIdx, _ := strconv.Atoi(ami)

	if err != nil || amiIdx >= len(validImages) {
		return nil, err
	}

	amiName := *validImages[amiIdx].Name
	amiId := *validImages[amiIdx].ImageId
	//Parse out version number
	amiNameInfo := strings.Split(amiName, "-")
	amiVersion := amiNameInfo[2]
	if app.Name == "taimurain" {
		amiVersion = amiNameInfo[1]
	}

	selectedAmi := SelectedAmi{
		Id:      amiId,
		Name:    amiName,
		Version: amiVersion,
	}

	return &selectedAmi, nil
}

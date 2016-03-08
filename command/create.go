package command

import (
	"bytes"
	"encoding/base64"
	"crypto/sha1"
	"io"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/mitchellh/cli"
	"sort"
	"strconv"
	"strings"
	"time"
)

type suripuApp struct {
	name                  string
	sg                    string
	instanceType          string
	instanceProfile       string
	keyName               string
	targetDesiredCapacity int64 //This is the desired capacity of the asg targeted for deployment
	usesPacker            bool
	javaVersion           int
	packagePath			  string
}

//This hash should be updated anytime default_userdata.sh is updated on S3
var expectedUserDataHash = "af45382d7e42b97a15708cc615a67b879cbabd9e"

var suripuApps []suripuApp = []suripuApp{
	suripuApp{
		name:                  "suripu-app",
		sg:                    "sg-d28624b6",
		instanceType:          "m3.medium",
		instanceProfile:       "suripu-app",
		targetDesiredCapacity: 2,
		usesPacker:            true,
		javaVersion:           7,
		packagePath:		   "com/hello/suripu"},
	suripuApp{
		name:                  "suripu-service",
		sg:                    "sg-11ac0e75",
		instanceType:          "m3.medium",
		instanceProfile:       "suripu-service",
		targetDesiredCapacity: 4,
		usesPacker:            false,
		javaVersion:           7,
		packagePath:		   "com/hello/suripu"},
	suripuApp{
		name:                  "suripu-workers",
		sg:                    "sg-7054d714",
		instanceType:          "c3.xlarge",
		instanceProfile:       "suripu-workers",
		targetDesiredCapacity: 2,
		usesPacker:            false,
		javaVersion:           7,
		packagePath:		   "com/hello/suripu"},
	suripuApp{
		name:                  "suripu-admin",
		sg:                    "sg-71773a16",
		instanceType:          "t2.micro",
		instanceProfile:       "suripu-admin",
		targetDesiredCapacity: 1,
		usesPacker:            false,
		javaVersion:           7,
		packagePath:		   "com/hello/suripu"},
	suripuApp{
		name:                  "logsindexer",
		sg:                    "sg-36f95050",
		instanceType:          "m3.medium",
		instanceProfile:       "logsindexer",
		targetDesiredCapacity: 1,
		usesPacker:            false,
		javaVersion:           8,
		packagePath:		   "com/hello/suripu"},
	suripuApp{
		name:                  "sense-firehose",
		sg:                    "sg-5296b834",
		instanceType:          "m3.medium",
		instanceProfile:       "sense-firehose",
		targetDesiredCapacity: 1,
		usesPacker:            false,
		javaVersion:           8,
		packagePath:		   "com/hello/suripu"},
	suripuApp{
		name:                  "hello-time",
		sg:                    "sg-5c371525",
		instanceType:          "t2.micro",
		instanceProfile:       "hello-time",
		targetDesiredCapacity: 1,
		usesPacker:            false,
		javaVersion:           7,
		packagePath:		   "com/hello/time"},
	suripuApp{
		name:                  "suripu-queue",
		sg:                    "sg-3e55ba46",
		instanceType:          "m3.medium",
		instanceProfile:       "suripu-workers",
		targetDesiredCapacity: 1,
		usesPacker:            true,
		javaVersion:           7,
		packagePath:		   "com/hello/suripu"},
	suripuApp{
		name:                  "messeji",
		sg:                    "sg-45c5c73c",
		instanceType:          "m3.medium",
		instanceProfile:       "messeji",
		targetDesiredCapacity: 4,
		usesPacker:            false,
		javaVersion:           8,
		packagePath:		   		 "com/hello"},
}

var keyBucket string = "hello-keys"

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

type CreateCommand struct {
	Ui       cli.ColoredUi
	Notifier BasicNotifier
}

func (c *CreateCommand) Help() string {
	helpText := `Usage: create`
	return strings.TrimSpace(helpText)
}

func (c *CreateCommand) Run(args []string) int {
	config := &aws.Config{
		Region: aws.String("us-east-1"),
	}
	s3KeyConfig := &aws.Config{
		Region: aws.String("us-west-1"),
	}
	asgService := autoscaling.New(session.New(), config)
	s3Service := s3.New(session.New(), config)
	s3KeyService := s3.New(session.New(), s3KeyConfig)
	ec2Service := ec2.New(session.New(), config)

	c.Ui.Output("Which app are we building for?")

	for idx, app := range suripuApps {
		c.Ui.Output(fmt.Sprintf("[%d] %s", idx, app.name))
	}

	appSel, err := c.Ui.Ask("Select an app #: ")
	appIdx, _ := strconv.Atoi(appSel)

	if err != nil || appIdx >= len(suripuApps) {
		c.Ui.Error(fmt.Sprintf("Incorrect app selection: %s\n", err))
		return 1
	}

	selectedApp := suripuApps[appIdx]

	var params *autoscaling.DescribeAccountLimitsInput
	accountLimits, err := asgService.DescribeAccountLimits(params)

	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	maxLCs := *accountLimits.MaxNumberOfLaunchConfigurations

	lcParams := &autoscaling.DescribeLaunchConfigurationsInput{
		MaxRecords: aws.Int64(100),
	}
	descLCs, err := asgService.DescribeLaunchConfigurations(lcParams)
	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		c.Ui.Error(err.Error())
		return 1
	}

	currentLCCount := len(descLCs.LaunchConfigurations)
	c.Ui.Info(fmt.Sprintf("Current Launch Config Capacity: %d/%d", currentLCCount, maxLCs))

	amiId := ""
	amiName := ""
	amiVersion := ""
	userData := ""

	if selectedApp.usesPacker {
		//Allow user to enter version number and search for AMI based on that
		c.Ui.Warn(fmt.Sprintf("%s not yet handled by Packer-free deployment. Proceeding with Packer-created AMI selection.", selectedApp.name))

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
		respAll, err := ec2Service.DescribeImages(ec2ParamsAll)

		if err != nil {
			c.Ui.Error(fmt.Sprintln(err.Error()))
			return 1
		}

		validImages := make([]*ec2.Image, 0)

		for _, image := range respAll.Images {
			if strings.HasPrefix(*image.Name, selectedApp.name) {
				validImages = append(validImages, image)
			}
		}

		sort.Sort(sort.Reverse(ByImageTime(validImages)))

		c.Ui.Output("Which AMI should be used?")
		numImages := Min(len(validImages), 10)
		for idx := 0; idx < numImages; idx++ {
			c.Ui.Output(fmt.Sprintf("[%d] \t%s\t%s", idx, *validImages[idx].Name, *validImages[idx].CreationDate))
		}

		ami, err := c.Ui.Ask("Select an AMI #: ")
		amiIdx, _ := strconv.Atoi(ami)

		if err != nil || amiIdx >= len(validImages) {
			c.Ui.Error(fmt.Sprintf("Incorrect AMI selection: %s\n", err))
			return 1
		}

		amiName = *validImages[amiIdx].Name
		amiId = *validImages[amiIdx].ImageId
		//Parse out version number
		amiNameInfo := strings.Split(amiName, "-")
		amiVersion = amiNameInfo[2]

	} else {

		pkgPrefix := fmt.Sprintf("packages/%s/%s/", selectedApp.packagePath, selectedApp.name)

		c.Ui.Output("")
		//retrieve package list from S3 for selectedApp
		s3ListParams := &s3.ListObjectsInput{
			Bucket: aws.String("hello-deploy"), // Required
			//Delimiter:    aws.String("/"),
			//EncodingType: aws.String("EncodingType"),
			//Marker:       aws.String("Marker"),
			MaxKeys: aws.Int64(100000),
			Prefix:  aws.String(pkgPrefix),
		}
		s3Resp, err := s3Service.ListObjects(s3ListParams)

		if err != nil {
			c.Ui.Error(fmt.Sprintln(err.Error()))
			return 0
		}

		availablePackages := make([]*s3.Object, 0)
		for _, item := range s3Resp.Contents {
			if strings.HasSuffix(*item.Key, ".deb") {
				availablePackages = append(availablePackages, item)
			}
		}
		sort.Sort(sort.Reverse(ByObjectLastModified(availablePackages)))

		versions := make([]string, 0)

		c.Ui.Info(fmt.Sprintf("Latest 10 packages available for %s:", selectedApp.name))
		c.Ui.Info(" #\tVersion:  \tLast Modified:")
		c.Ui.Info("---|----------------|---------------")
		numImages := Min(len(availablePackages), 10)
		for idx := 0; idx < numImages; idx++ {
			objectKeyChunks := strings.Split(*availablePackages[idx].Key, "/")
			versionNumber := objectKeyChunks[len(objectKeyChunks)-2]
			versions = append(versions, versionNumber)
			c.Ui.Output(fmt.Sprintf("[%d]\t%s\t\t%s", idx, versionNumber, availablePackages[idx].LastModified.Format(time.UnixDate)))
		}

		ver, err := c.Ui.Ask("Select a version #: ")
		verIdx, _ := strconv.Atoi(ver)

		if err != nil || verIdx >= len(availablePackages) {
			c.Ui.Error(fmt.Sprintf("Incorrect version selection: %s\n", err))
			return 1
		}

		amiVersion = versions[verIdx]

		//Get the userdata template from S3 for instance startup using cloud-init
		s3params := &s3.GetObjectInput{
			Bucket: aws.String("hello-deploy"),                 // Required
			Key:    aws.String("userdata/default_userdata.sh"), // Required
		}
		resp, err := s3Service.GetObject(s3params)

		if err != nil {
			c.Ui.Error(fmt.Sprintln(err.Error()))
			return 0
		}

		// Pretty-print the response data.
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		userData = buf.String()

		//Verify checksum of userData
		hash := sha1.New()
		io.WriteString(hash, userData)
		userDataHash := fmt.Sprintf("%x", hash.Sum(nil))

		if userDataHash != expectedUserDataHash {
			c.Ui.Error("UserData hash from S3 does not match the one expected by your version of Sanders. Please correct this before proceeding.")
			return 0
		}

		//do token replacement
		userData = strings.Replace(userData, "{app_version}", amiVersion, -1)
		userData = strings.Replace(userData, "{app_name}", selectedApp.name, -1)
		userData = strings.Replace(userData, "{package_path}", selectedApp.packagePath, -1)
		userData = strings.Replace(userData, "{default_region}", "us-east-1", -1)
		userData = strings.Replace(userData, "{java_version}", strconv.Itoa(selectedApp.javaVersion), -1)

		userData = base64.StdEncoding.EncodeToString([]byte(userData))

		amiName = "a cloud-init deploy based on the AMI: Base-2016-03-08"
		amiId = "ami-d06267ba"
	}

	c.Ui.Info(fmt.Sprintf("You selected %s\n", amiName))
	c.Ui.Info(fmt.Sprintf("Version Number: %s\n", amiVersion))

	launchConfigName := fmt.Sprintf("%s-prod-%s", selectedApp.name, amiVersion)

	//Create deployment-specific KeyPair

	keyName := fmt.Sprintf("%s-%d", launchConfigName, time.Now().Unix())
	keyPairParams := &ec2.CreateKeyPairInput{
		KeyName: aws.String(keyName), // Required
		DryRun:  aws.Bool(false),
	}
	keyPairResp, err := ec2Service.CreateKeyPair(keyPairParams)

	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to create KeyPair. %s", err.Error()))
		return 1
	}

	c.Ui.Info(fmt.Sprintf("Created KeyPair: %s. \n", *keyPairResp.KeyName))

	//Upload key to S3
	key := fmt.Sprintf("/prod/%s/%s.pem", selectedApp.name, *keyPairResp.KeyName)

	uploadResult, err := s3KeyService.PutObject(&s3.PutObjectInput{
		Body:   	strings.NewReader(*keyPairResp.KeyMaterial),
		Bucket:		aws.String(keyBucket),
		Key:    	&key,
	})

	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to upload key %s. %s\n", key, err))
		return 1
	}

	c.Ui.Info(fmt.Sprintf("Uploaded key %s to S3. (Etag: %s)\n", key, *uploadResult.ETag))

	createLCParams := &autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName:  aws.String(launchConfigName), // Required
		AssociatePublicIpAddress: aws.Bool(true),
		IamInstanceProfile:       aws.String(selectedApp.instanceProfile),
		ImageId:                  aws.String(amiId),
		InstanceMonitoring: &autoscaling.InstanceMonitoring{
			Enabled: aws.Bool(true),
		},
		InstanceType:     aws.String(selectedApp.instanceType),
		KeyName:          aws.String(keyName),
		SecurityGroups: []*string{
			aws.String(selectedApp.sg), // Required
		},
		UserData: aws.String(userData),
	}

	deployAction := NewDeployAction("create", selectedApp.name, launchConfigName, 0)

	c.Ui.Info(fmt.Sprint("Creating Launch Configuration with the following parameters:"))
	c.Ui.Info(fmt.Sprint(createLCParams))
	ok, err := c.Ui.Ask("'ok' if you agree, anything else to cancel: ")
	if err != nil {
		c.Ui.Error(fmt.Sprintf("%s", err))
		Cleanup(keyName, key, c.Ui)
		return 1
	}

	if ok != "ok" {
		c.Ui.Warn("Cancelled.")
		if !Cleanup(keyName, key, c.Ui) {
			return 1
		}
		return 0
	}

	_, createError := asgService.CreateLaunchConfiguration(createLCParams)

	if createError != nil {
		// Message from an error.
		c.Ui.Error(fmt.Sprintf("Failed to create Launch Configuration: %s", launchConfigName))
		c.Ui.Error(fmt.Sprintln(createError.Error()))
		Cleanup(keyName, key, c.Ui)
		return 1
	}

	c.Notifier.Notify(deployAction)
	c.Ui.Output(fmt.Sprintln("Launch Configuration created."))

	return 0
}

func Cleanup(keyName string, objectName string, ui cli.ColoredUi ) bool {

	ui.Info("")
	ui.Info(fmt.Sprintf("Cleaning up created KeyPair: %s", keyName))

	//Remove any created keys from S3 & EC2
	ec2Service := ec2.New(session.New(), aws.NewConfig().WithRegion("us-east-1"))
	s3KeyService := s3.New(session.New(), aws.NewConfig().WithRegion("us-west-1"))

	//Delete key from EC2
	params := &ec2.DeleteKeyPairInput{
		KeyName: aws.String(keyName), // Required
		DryRun:  aws.Bool(false),
	}
	_, err := ec2Service.DeleteKeyPair(params)

	if err != nil {
		ui.Error(fmt.Sprintf("Failed to delete KeyPair: %s", err.Error()))
		return false
	}

	ui.Info(fmt.Sprintf("Successfully deleted KeyPair: %s", keyName))

	//Delete pem file from s3
	delParams := &s3.DeleteObjectInput{
		Bucket:       aws.String(keyBucket), // Required
		Key:          aws.String(objectName),  // Required
	}
	_, objErr := s3KeyService.DeleteObject(delParams)

	if objErr != nil {
		ui.Error(fmt.Sprintf("Failed to delete S3 Object: %s", objErr.Error()))
		return false
	}

	ui.Info(fmt.Sprintf("Successfully deleted S3 object: %s", objectName))

	return true
}

func (c *CreateCommand) Synopsis() string {
	return "Creates a launch configuration based on selected parameters. (Only for boxfuse-created AMIs)"
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

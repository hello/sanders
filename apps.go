package main

import (
	"github.com/hello/sanders/core"
)

var suripuApps = []core.SuripuApp{
	{
		Name:                  "suripu-app",
		SecurityGroup:         "sg-d28624b6",
		InstanceType:          "t2.medium",
		InstanceProfile:       "suripu-app",
		TargetDesiredCapacity: 2,
		UsesPacker:            false,
		JavaVersion:           8,
		PackagePath:           "com/hello/suripu",
	},
	{
		Name:                  "suripu-service",
		SecurityGroup:         "sg-11ac0e75",
		InstanceType:          "t2.medium",
		InstanceProfile:       "suripu-service",
		TargetDesiredCapacity: 3,
		UsesPacker:            false,
		JavaVersion:           8,
		PackagePath:           "com/hello/suripu",
	},
	{
		Name:                  "suripu-workers",
		SecurityGroup:         "sg-7054d714",
		InstanceType:          "c3.xlarge",
		InstanceProfile:       "suripu-workers",
		TargetDesiredCapacity: 2,
		UsesPacker:            false,
		JavaVersion:           8,
		PackagePath:           "com/hello/suripu",
		Spot: &core.SpotSettings{
			Price: "0.210",
		},
	},
	{
		Name:                  "suripu-admin",
		SecurityGroup:         "sg-71773a16",
		InstanceType:          "t2.micro",
		InstanceProfile:       "suripu-admin",
		TargetDesiredCapacity: 1,
		UsesPacker:            false,
		JavaVersion:           8,
		PackagePath:           "com/hello/suripu",
	},
	{
		Name:                  "logsindexer",
		SecurityGroup:         "sg-36f95050",
		InstanceType:          "t2.micro",
		InstanceProfile:       "logsindexer",
		TargetDesiredCapacity: 1,
		UsesPacker:            false,
		JavaVersion:           8,
		PackagePath:           "com/hello/suripu",
	},
	{
		Name:                  "sense-firehose",
		SecurityGroup:         "sg-5296b834",
		InstanceType:          "m3.medium",
		InstanceProfile:       "sense-firehose",
		TargetDesiredCapacity: 1,
		UsesPacker:            false,
		JavaVersion:           8,
		PackagePath:           "com/hello/suripu",
	},
	{
		Name:                  "hello-time",
		SecurityGroup:         "sg-5c371525",
		InstanceType:          "t2.nano",
		InstanceProfile:       "hello-time",
		TargetDesiredCapacity: 2,
		UsesPacker:            false,
		JavaVersion:           8,
		PackagePath:           "com/hello/time",
	},
	{
		Name:                  "suripu-queue",
		SecurityGroup:         "sg-3e55ba46",
		InstanceType:          "c3.large",
		InstanceProfile:       "suripu-queue",
		TargetDesiredCapacity: 1,
		UsesPacker:            false,
		JavaVersion:           8,
		PackagePath:           "com/hello/suripu",
	},
	{
		Name:                  "messeji",
		SecurityGroup:         "sg-45c5c73c",
		InstanceType:          "m3.medium",
		InstanceProfile:       "messeji",
		TargetDesiredCapacity: 2,
		UsesPacker:            false,
		JavaVersion:           8,
		PackagePath:           "com/hello",
	},
	{
		Name:                  "taimurain",
		SecurityGroup:         "sg-b3f631c8",
		InstanceType:          "c4.xlarge",
		InstanceProfile:       "taimurain",
		TargetDesiredCapacity: 3,
		UsesPacker:            true,
		JavaVersion:           8,
		PackagePath:           "com/hello",
	},
}

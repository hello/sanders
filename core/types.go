package core

type SelectedAmi struct {
	Id       string
	Name     string
	Version  string
	UserData string
}

type SpotSettings struct {
	Price string
}
type SuripuApp struct {
	Name                  string
	SecurityGroup         string
	InstanceType          string
	InstanceProfile       string
	KeyName               string
	TargetDesiredCapacity int64 //This is the desired capacity of the asg targeted for deployment
	UsesPacker            bool
	JavaVersion           int
	PackagePath           string
	Spot                  *SpotSettings
}

type Tag struct {
	AsgName   string
	TagName   string
	TagValue  string
	Propagate bool
}

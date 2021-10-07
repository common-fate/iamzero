package audit

// CfnTemplate is the YAML CloudFormation template
type CfnTemplate struct {
	Conditions interface{}
	Resources  map[string]CfnResource `yaml:"Resources"`
}

type CfnResource struct {
	Metadata struct {
		AwsCdkPath string `yaml:"aws:cdk:path"`
	} `yaml:"Metadata"`
	Properties interface{}
	Type       string `yaml:"Type"`
}

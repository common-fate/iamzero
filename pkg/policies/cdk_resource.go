package policies

type CDKResource struct {
	// the ID of the CloudFormation stack the resource is defined in
	StackID    string `json:"stackId"`
	LogicalID  string `json:"logicalId"`
	PhysicalID string `json:"physicalId"`
	AccountID  string `json:"accountId"`
	// The full path to the resource in the CDK stack e.g. CdkExampleStack/iamzero-example-role/Resource
	CDKPath string `json:"cdkPath"`
	// The ID of the resource in the CDK stack e.g. iamzero-example-role
	CDKID string `json:"cdkId"`
	Type  string `json:"type"`
}

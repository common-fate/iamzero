package recommendations

// CDKFinding is proposed CDK source code changes recommended by IAM Zero
type CDKFinding struct {
	FindingID       string              `json:"findingId"`
	Role            CDKRole             `json:"role"`
	Recommendations []CDKRecommendation `json:"recommendations"`
}

// CDKRole is a reference to a user or role defined in CDK
type CDKRole struct {
	Type    string `json:"type"`
	CDKPath string `json:"cdkPath"`
}

type CDKRecommendation struct {
	Type       string         `json:"type"`
	Statements []CDKStatement `json:"statements"`
}

type CDKStatement struct {
	Resources []CDKResource `json:"resources"`
	Actions   []string      `json:"actions"`
}

type CDKResource struct {
	Reference string  `json:"reference"`
	Type      string  `json:"type"`
	CDKPath   *string `json:"cdkPath,omitempty"`
	ARN       *string `json:"arn,omitempty"`
}

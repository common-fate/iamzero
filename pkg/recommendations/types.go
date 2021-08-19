package recommendations

import "github.com/common-fate/iamzero/pkg/audit"

const (
	AlertActive   = "active"
	AlertIgnored  = "ignored"
	AlertFixed    = "fixed"
	AlertApplying = "applying"
)

// AWSEvent is an API call logged by an AWS SDK
// instrumented with iamzero
type AWSEvent struct {
	Time     string      `json:"time"`
	Data     AWSData     `json:"data"`
	Identity AWSIdentity `json:"identity"`
}

type AWSData struct {
	Type             string                 `json:"type"`
	Service          string                 `json:"service"`
	Region           string                 `json:"region"`
	Operation        string                 `json:"operation"`
	Parameters       map[string]interface{} `json:"parameters"`
	ExceptionMessage string                 `json:"exceptionMessage"`
	ExceptionCode    string                 `json:"exceptionCode"`
}

type AWSIdentity struct {
	User    string `json:"user"`
	Role    string `json:"role"`
	Account string `json:"account"`
}
type Advisor struct {
	AlertsMapping map[string][]AdvisoryTemplate
	auditor       *audit.Auditor
}

type Description struct {
	AppliedTo string
	Type      string
	Policy    interface{}
}

type RecommendationDetails struct {
	ID          string
	Comment     string
	Resources   []Resource
	Description []Description
}

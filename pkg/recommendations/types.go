package recommendations

import (
	"github.com/common-fate/iamzero/pkg/audit"

	"github.com/mitchellh/hashstructure/v2"
)

const (
	AlertActive   = "active"
	AlertIgnored  = "ignored"
	AlertFixed    = "fixed"
	AlertApplying = "applying"
)

// AWSEvent is an API call logged by an AWS SDK
// instrumented with iamzero
type AWSEvent struct {
	ID       string      `json:"id" hash:"ignore"`
	Time     string      `json:"time" hash:"ignore"`
	Data     AWSData     `json:"data"`
	Identity AWSIdentity `json:"identity"`
}

// HashEvent calculates a hash of the event so that we can deduplicate it.
// We specifically don't hash the time by applying a `hash:"ignore"`
// tag to the Time field in the AWSEvent struct
func HashEvent(e AWSEvent) (uint64, error) {
	return hashstructure.Hash(e, hashstructure.FormatV2, nil)
}

type AWSData struct {
	// Type is either "awsAction" or "awsError"
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
	Resources   []CloudResourceInstance
	Description []Description
}

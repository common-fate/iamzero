package recommendations

import (
	"go.uber.org/zap"
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
	Time     string      `json:"time"`
	Data     AWSData     `json:"data"`
	Identity AWSIdentity `json:"identity"`
}

type AWSData struct {
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

type AWSIAMPolicy struct {
	Version   string
	Id        *string
	Statement []AWSIAMStatement
}

type AWSIAMStatement struct {
	Sid       string
	Effect    string
	Action    []string
	Principal *AWSIAMPrincipal `json:",omitempty"`
	Resource  []string
}

type AWSIAMPrincipal struct {
	AWS string
}

// AdviceFactory generates Advice based on a provided event
type AdviceFactory = func(e AWSEvent) (Advice, error)

type Advisor struct {
	AlertsMapping map[string][]AdviceFactory
}

type Description struct {
	AppliedTo string
	Type      string
	Policy    interface{}
}

type RecommendationDetails struct {
	ID          string
	Comment     string
	Description []Description
}

type Advice interface {
	Apply(log *zap.SugaredLogger) error
	GetID() string
	Details() RecommendationDetails
}

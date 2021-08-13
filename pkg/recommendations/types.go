package recommendations

import (
	"encoding/json"
	"errors"
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

type AWSIAMPolicy struct {
	Version   string
	Id        *string `json:",omitempty"`
	Statement IAMStatements
}

type AWSIAMStatement struct {
	Sid       string
	Effect    string
	Action    StringOrStringArray
	Principal *AWSIAMPrincipal `json:",omitempty"`
	Resource  StringOrStringArray
}

// IAMStatements implements a custom UnmarshalJSON to handle
// cases where AWS returns a single statement with no enclosing
// array
type IAMStatements []AWSIAMStatement

func (s *IAMStatements) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return errors.New("no bytes to unmarshal")
	}
	switch data[0] {
	case '{':
		return s.unmarshalSingle(data)
	case '[':
		return s.unmarshalMany(data)
	}
	return errors.New("unmarshalling: AWS IAM statement neither struct nor array")
}

func (s *IAMStatements) unmarshalSingle(data []byte) error {
	var res AWSIAMStatement
	err := json.Unmarshal(data, &res)
	if err != nil {
		return err
	}
	*s = []AWSIAMStatement{res}
	return nil
}

func (s *IAMStatements) unmarshalMany(data []byte) error {
	var res []AWSIAMStatement
	err := json.Unmarshal(data, &res)
	if err != nil {
		return err
	}
	*s = res
	return nil
}

// StringOrStringArray allows AWS statements to be unmarshalled
// into a string slice.
// Sometimes AWS policy JSONs contain a string, rather than a string array.
type StringOrStringArray []string

func (c *StringOrStringArray) UnmarshalJSON(data []byte) error {
	var tmp interface{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	slice, ok := tmp.([]interface{})
	if ok {
		for _, item := range slice {
			*c = append(*c, item.(string))
		}
		return nil
	}
	theString, ok := tmp.(string)
	if ok {
		*c = append(*c, theString)
		return nil
	}
	return errors.New("Field neither slice or string")
}

type AWSIAMPrincipal struct {
	AWS string
}

// AdviceFactory generates Advice based on a provided event
type AdviceFactory = func(e AWSEvent) (*JSONAdvice, error)

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
	Resources   []Resource
	Description []Description
}

package recommendations

import (
	"time"

	"github.com/common-fate/iamzero/pkg/tokens"
)

// Policy is a least-privilege policy generated by IAM Zero
type Policy struct {
	ID          string       `json:"id"`
	Identity    AWSIdentity  `json:"identity"`
	LastUpdated time.Time    `json:"lastUpdated"`
	Token       tokens.Token `json:"token"`
	EventCount  int          `json:"eventCount"`
	Document    AWSIAMPolicy `json:"document"`
	// Status is either "active" or "dismissed"
	Status string `json:"status"`
}

// RecalculateDocument rebuilds the policy document based on the actions
// this initial implementation is naive and doesn't deduplicate or aggregate policies.
func (p *Policy) RecalculateDocument(actions []AWSAction) {
	statements := []AWSIAMStatement{}

	for _, alert := range actions {
		if alert.Enabled && len(alert.Recommendations) > 0 {
			advisory := alert.GetSelectedAdvisory()
			for _, description := range advisory.Details().Description {
				// TODO: this should be redesigned to avoid casting from the interface.
				policy, ok := description.Policy.(AWSIAMPolicy)
				if ok {
					statements = append(statements, policy.Statement...)
				}
			}
		}
	}

	p.LastUpdated = time.Now()
	p.EventCount = len(actions)
	p.Document.Statement = statements
}

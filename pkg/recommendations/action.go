package recommendations

import (
	"errors"
	"time"

	"github.com/common-fate/iamzero/pkg/policies"
)

// Resource is a cloud resource such as an S3 bucket which permissions can be granted for
// Currently we just use this in the UI to display a human-friendly list of resources
// for each recorded action.
type Resource struct {
	ID string `json:"id"`
	// a friendly name for the resource
	Name        string                `json:"name"`
	CDKResource *policies.CDKResource `json:"cdkResource"`
}

type AWSAction struct {
	ID                 string        `json:"id"`
	PolicyID           string        `json:"policyId"`
	Event              AWSEvent      `json:"event"`
	Status             string        `json:"status"`
	Time               time.Time     `json:"time"`
	Resources          []Resource    `json:"resources"`
	Recommendations    []*JSONAdvice `json:"recommendations"`
	HasRecommendations bool          `json:"hasRecommendations"`
	// Enabled indicates whether this action is used in a least-privilege policy
	Enabled bool `json:"enabled"`
	// SelectedAdvisoryID is the ID of the advisory selected by the user to resolve the policy
	SelectedAdvisoryID string `json:"selectedAdvisoryId"`
}

// SelectAdvisory sets the `SelectedAdvisoryID` field.
// Returns an error if the advisory ID does not exist in `Recommendations`
func (a *AWSAction) SelectAdvisory(id string) error {
	for _, r := range a.Recommendations {
		if r.GetID() == id {
			a.SelectedAdvisoryID = id
			return nil
		}
	}
	return errors.New("could not find advisory")
}

// GetSelectedAdvisory returns the Advice object matching the action's SelectedAdvisoryID
func (a *AWSAction) GetSelectedAdvisory() *JSONAdvice {
	for _, r := range a.Recommendations {
		if r.GetID() == a.SelectedAdvisoryID {
			return r
		}
	}
	return nil
}

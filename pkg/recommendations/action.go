package recommendations

import (
	"errors"
	"time"
)

type AWSAction struct {
	ID                 string    `json:"id"`
	PolicyID           string    `json:"policyId"`
	Event              AWSEvent  `json:"event"`
	Status             string    `json:"status"`
	Time               time.Time `json:"time"`
	Recommendations    []Advice  `json:"recommendations"`
	HasRecommendations bool      `json:"hasRecommendations"`
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
func (a *AWSAction) GetSelectedAdvisory() Advice {
	for _, r := range a.Recommendations {
		if r.GetID() == a.SelectedAdvisoryID {
			return r
		}
	}
	return nil
}

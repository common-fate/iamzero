package storage

import (
	"errors"

	"github.com/common-fate/iamzero/pkg/recommendations"
)

type ActionStorage struct {
	actions []recommendations.AWSAction
}

func NewAlertStorage() ActionStorage {
	return ActionStorage{actions: []recommendations.AWSAction{}}
}

func (a *ActionStorage) Add(action recommendations.AWSAction) {
	a.actions = append(a.actions, action)
}

func (a *ActionStorage) List() []recommendations.AWSAction {
	return a.actions
}

func (a *ActionStorage) Get(id string) *recommendations.AWSAction {
	for _, action := range a.actions {
		if action.ID == id {
			return &action
		}
	}
	return nil
}

// ListForPolicy lists all the actions that related to a given policy
func (a *ActionStorage) ListForPolicy(policyID string) []recommendations.AWSAction {
	actions := []recommendations.AWSAction{}

	for _, action := range a.actions {
		if action.PolicyID == policyID {
			actions = append(actions, action)
		}
	}
	return actions
}

func (a *ActionStorage) SetStatus(id string, status string) error {
	for i, alert := range a.actions {
		if alert.ID == id {
			a.actions[i].Status = status
			return nil
		}
	}
	return errors.New("could not find alert")
}

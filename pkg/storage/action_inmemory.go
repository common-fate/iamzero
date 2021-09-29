package storage

import (
	"errors"
	"sync"

	"github.com/common-fate/iamzero/pkg/recommendations"
)

type InMemoryActionStorage struct {
	sync.RWMutex
	actions []recommendations.AWSAction
}

func (a *InMemoryActionStorage) ListEnabledActionsForPolicy(policyID string) ([]recommendations.AWSAction, error) {
	actions, err := a.ListForPolicy(policyID)
	if err != nil {
		return nil, err
	}
	var enabledActions []recommendations.AWSAction
	for _, a := range actions {
		if a.Enabled {
			enabledActions = append(enabledActions, a)
		}
	}
	return enabledActions, nil
}

func NewInMemoryActionStorage() *InMemoryActionStorage {
	return &InMemoryActionStorage{actions: []recommendations.AWSAction{}}
}

func (a *InMemoryActionStorage) Add(action recommendations.AWSAction) error {
	a.Lock()
	defer a.Unlock()
	a.actions = append(a.actions, action)
	return nil
}

func (a *InMemoryActionStorage) List() ([]recommendations.AWSAction, error) {
	return a.actions, nil
}

func (a *InMemoryActionStorage) Get(id string) (*recommendations.AWSAction, error) {
	for _, action := range a.actions {
		if action.ID == id {
			return &action, nil
		}
	}
	return nil, nil
}

// ListForPolicy lists all the actions that related to a given policy
func (a *InMemoryActionStorage) ListForPolicy(policyID string) ([]recommendations.AWSAction, error) {
	actions := []recommendations.AWSAction{}

	for _, action := range a.actions {
		if action.PolicyID == policyID {
			actions = append(actions, action)
		}
	}
	return actions, nil
}

func (a *InMemoryActionStorage) SetStatus(id string, status string) error {
	a.Lock()
	defer a.Unlock()
	for i, alert := range a.actions {
		if alert.ID == id {
			a.actions[i].Status = status
			return nil
		}
	}
	return errors.New("could not find alert")
}

func (s *InMemoryActionStorage) Update(action recommendations.AWSAction) error {
	s.Lock()
	defer s.Unlock()
	for i, a := range s.actions {
		if a.ID == action.ID {
			s.actions[i] = action
			return nil
		}
	}
	return errors.New("could not find alert")
}

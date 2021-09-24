package storage

import (
	"github.com/asdine/storm/v3"
	"github.com/common-fate/iamzero/pkg/recommendations"
)

type BoltActionStorage struct {
	db *storm.DB
}

func NewBoltActionStorage(db *storm.DB) *BoltActionStorage {
	return &BoltActionStorage{db: db}
}

func (a *BoltActionStorage) Add(action recommendations.AWSAction) error {
	return a.db.Save(&action)
}

func (a *BoltActionStorage) List() ([]recommendations.AWSAction, error) {
	var actions []recommendations.AWSAction

	err := a.db.All(&actions)
	if err != nil {
		return nil, err
	}
	return actions, nil
}

func (a *BoltActionStorage) Get(id string) (*recommendations.AWSAction, error) {
	var action recommendations.AWSAction

	err := a.db.One("ID", id, &action)
	if err != nil {
		return nil, err
	}
	return &action, nil
}

// ListForPolicy lists all the actions that related to a given policy
//
// Can filter for only enabled policies
func (a *BoltActionStorage) ListForPolicy(policyID string) ([]recommendations.AWSAction, error) {
	actions, err := a.List()
	if err != nil {
		return nil, err
	}

	actionsForPolicy := []recommendations.AWSAction{}
	for _, action := range actions {
		if action.PolicyID == policyID {
			actionsForPolicy = append(actionsForPolicy, action)

		}
	}

	return actionsForPolicy, nil
}

func (a *BoltActionStorage) SetStatus(id string, status string) error {
	action, err := a.Get(id)
	if err != nil {
		return err
	}
	action.Status = status
	return a.Update(*action)
}

func (s *BoltActionStorage) Update(action recommendations.AWSAction) error {
	return s.db.Save(&action)
}

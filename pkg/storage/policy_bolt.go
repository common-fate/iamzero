package storage

import (
	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/pkg/errors"

	"github.com/asdine/storm/v3"
)

type BoltPolicyStorage struct {
	db *storm.DB
}

func NewBoltPolicyStorage(db *storm.DB) *BoltPolicyStorage {
	return &BoltPolicyStorage{db: db}
}

func (s *BoltPolicyStorage) List() ([]recommendations.Policy, error) {
	policies := []recommendations.Policy{}

	err := s.db.All(&policies)
	if err != nil {
		return nil, errors.Wrap(err, "boltdb list policies")
	}

	return policies, err
}

func (s *BoltPolicyStorage) ListForStatus(status string) ([]recommendations.Policy, error) {
	policies := []recommendations.Policy{}

	err := s.db.Find("Status", status, &policies)
	if err != nil {
		return []recommendations.Policy{}, nil
	}

	return policies, nil
}

func (s *BoltPolicyStorage) Get(id string) (*recommendations.Policy, error) {
	var p recommendations.Policy

	err := s.db.One("ID", id, &p)
	if err != nil {
		return nil, errors.Wrap(err, "boltdb get policy")
	}
	return &p, err
}

// FindByRole finds a matching policy by its role
func (s *BoltPolicyStorage) FindByRole(query FindByRoleQuery) (*recommendations.Policy, error) {
	policies := []recommendations.Policy{}

	err := s.db.All(&policies)
	if err != nil {
		return nil, err
	}

	for _, policy := range policies {
		if policy.Identity.Role == query.Role && policy.Status == query.Status {
			return &policy, nil
		}
	}
	return nil, nil
}

func (s *BoltPolicyStorage) CreateOrUpdate(policy recommendations.Policy) error {
	return s.db.Save(&policy)
}

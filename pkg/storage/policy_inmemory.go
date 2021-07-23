package storage

import (
	"github.com/common-fate/iamzero/pkg/recommendations"
)

type PolicyStorage struct {
	policies []recommendations.Policy
}

func NewPolicyStorage() PolicyStorage {
	return PolicyStorage{policies: []recommendations.Policy{}}
}

func (s *PolicyStorage) List() []recommendations.Policy {
	return s.policies
}

func (s *PolicyStorage) ListForStatus(status string) []recommendations.Policy {
	policies := []recommendations.Policy{}
	for _, p := range s.policies {
		if p.Status == status {
			policies = append(policies, p)
		}
	}
	return policies
}

func (s *PolicyStorage) Get(id string) *recommendations.Policy {
	for _, policy := range s.policies {
		if policy.ID == id {
			return &policy
		}
	}
	return nil
}

type FindPolicyQuery struct {
	Role   string
	Token  string
	Status string
}

// FindByRoleAndToken finds a matching policy by its role and token
func (s *PolicyStorage) FindByRoleAndToken(q FindPolicyQuery) *recommendations.Policy {
	for _, policy := range s.policies {
		if policy.Identity.Role == q.Role && policy.Token.ID == q.Token && policy.Status == q.Status {
			return &policy
		}
	}
	return nil
}

func (s *PolicyStorage) CreateOrUpdate(policy recommendations.Policy) error {
	for i, p := range s.policies {
		if p.ID == policy.ID {
			s.policies[i] = policy
			return nil
		}
	}
	// add a new policy if it doesn't exist
	s.policies = append(s.policies, policy)
	return nil
}

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

func (s *PolicyStorage) Get(id string) *recommendations.Policy {
	for _, policy := range s.policies {
		if policy.ID == id {
			return &policy
		}
	}
	return nil
}

// FindByRoleAndToken finds a matching policy by its role and token
func (s *PolicyStorage) FindByRoleAndToken(role, token string) *recommendations.Policy {
	for _, policy := range s.policies {
		if policy.Identity.Role == role && policy.Token.ID == token {
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

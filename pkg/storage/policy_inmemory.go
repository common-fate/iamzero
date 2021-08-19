package storage

import (
	"sync"

	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/common-fate/iamzero/pkg/tokens"
)

type InMemoryPolicyStorage struct {
	sync.RWMutex
	policies []recommendations.Policy
}

func NewInMemoryPolicyStorage() *InMemoryPolicyStorage {
	return &InMemoryPolicyStorage{policies: []recommendations.Policy{}}
}

func (s *InMemoryPolicyStorage) List() ([]recommendations.Policy, error) {
	return s.policies, nil
}

func (s *InMemoryPolicyStorage) ListForStatus(status string) ([]recommendations.Policy, error) {
	policies := []recommendations.Policy{}
	for _, p := range s.policies {
		if p.Status == status {
			policies = append(policies, p)
		}
	}
	return policies, nil
}

func (s *InMemoryPolicyStorage) Get(id string) (*recommendations.Policy, error) {
	for _, policy := range s.policies {
		if policy.ID == id {
			return &policy, nil
		}
	}
	return nil, nil
}

type FindPolicyQuery struct {
	Role   string
	Token  *tokens.Token
	Status string
}

type FindByRoleQuery struct {
	Role   string
	Status string
}

// FindByRole finds a matching policy by its role
func (s *InMemoryPolicyStorage) FindByRole(q FindByRoleQuery) (*recommendations.Policy, error) {
	for _, policy := range s.policies {
		if policy.Identity.Role == q.Role && policy.Status == q.Status {
			return &policy, nil
		}
	}
	return nil, nil
}

func (s *InMemoryPolicyStorage) CreateOrUpdate(policy recommendations.Policy) error {
	s.Lock()
	defer s.Unlock()
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

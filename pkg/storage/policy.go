package storage

import "github.com/common-fate/iamzero/pkg/recommendations"

type PolicyStorage interface {
	List() ([]recommendations.Policy, error)
	ListForStatus(status string) ([]recommendations.Policy, error)
	Get(id string) (*recommendations.Policy, error)
	FindByRole(q FindByRoleQuery) (*recommendations.Policy, error)
	CreateOrUpdate(policy recommendations.Policy) error
}

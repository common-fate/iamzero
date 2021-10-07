package storage

import "github.com/common-fate/iamzero/pkg/recommendations"

type FindingStorage interface {
	List() ([]recommendations.Finding, error)
	ListForStatus(status string) ([]recommendations.Finding, error)
	Get(id string) (*recommendations.Finding, error)
	FindByRole(q FindByRoleQuery) (*recommendations.Finding, error)
	CreateOrUpdate(finding recommendations.Finding) error
}

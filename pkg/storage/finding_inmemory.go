package storage

import (
	"sync"

	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/common-fate/iamzero/pkg/tokens"
)

type InMemoryFindingStorage struct {
	sync.RWMutex
	findings []recommendations.Finding
}

func NewInMemoryFindingStorage() *InMemoryFindingStorage {
	return &InMemoryFindingStorage{findings: []recommendations.Finding{}}
}

func (s *InMemoryFindingStorage) List() ([]recommendations.Finding, error) {
	return s.findings, nil
}

func (s *InMemoryFindingStorage) ListForStatus(status string) ([]recommendations.Finding, error) {
	findings := []recommendations.Finding{}
	for _, f := range s.findings {
		if f.Status == status {
			findings = append(findings, f)
		}
	}
	return findings, nil
}

func (s *InMemoryFindingStorage) Get(id string) (*recommendations.Finding, error) {
	for _, finding := range s.findings {
		if finding.ID == id {
			return &finding, nil
		}
	}
	return nil, nil
}

type FindFindingQuery struct {
	Role   string
	Token  *tokens.Token
	Status string
}

type FindByRoleQuery struct {
	Role   string
	Status string
}

// FindByRole finds a matching finding by its role
func (s *InMemoryFindingStorage) FindByRole(q FindByRoleQuery) (*recommendations.Finding, error) {
	for _, finding := range s.findings {
		if finding.Identity.Role == q.Role && finding.Status == q.Status {
			return &finding, nil
		}
	}
	return nil, nil
}

func (s *InMemoryFindingStorage) CreateOrUpdate(finding recommendations.Finding) error {
	s.Lock()
	defer s.Unlock()
	for i, f := range s.findings {
		if f.ID == finding.ID {
			s.findings[i] = finding
			return nil
		}
	}
	// add a new policy if it doesn't exist
	s.findings = append(s.findings, finding)
	return nil
}

package storage

import (
	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/pkg/errors"

	"github.com/asdine/storm/v3"
)

type BoltFindingStorage struct {
	db *storm.DB
}

func NewBoltFindingStorage(db *storm.DB) *BoltFindingStorage {
	return &BoltFindingStorage{db: db}
}

func (s *BoltFindingStorage) List() ([]recommendations.Finding, error) {
	findings := []recommendations.Finding{}

	err := s.db.All(&findings)
	if err != nil {
		return nil, errors.Wrap(err, "boltdb list policies")
	}

	return findings, err
}

func (s *BoltFindingStorage) ListForStatus(status string) ([]recommendations.Finding, error) {
	findings := []recommendations.Finding{}

	err := s.db.Find("Status", status, &findings)
	if err != nil {
		return []recommendations.Finding{}, nil
	}

	return findings, nil
}

func (s *BoltFindingStorage) Get(id string) (*recommendations.Finding, error) {
	var p recommendations.Finding

	err := s.db.One("ID", id, &p)
	if err != nil {
		return nil, errors.Wrap(err, "boltdb get finding")
	}
	return &p, err
}

// FindByRole finds a matching finding by its role
func (s *BoltFindingStorage) FindByRole(query FindByRoleQuery) (*recommendations.Finding, error) {
	policies := []recommendations.Finding{}

	err := s.db.All(&policies)
	if err != nil {
		return nil, err
	}

	for _, finding := range policies {
		if finding.Identity.Role == query.Role && finding.Status == query.Status {
			return &finding, nil
		}
	}
	return nil, nil
}

func (s *BoltFindingStorage) CreateOrUpdate(finding recommendations.Finding) error {
	return s.db.Save(&finding)
}

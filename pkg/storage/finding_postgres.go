package storage

import (
	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type PostgresFindingStorage struct {
	db *sqlx.DB
}

func NewPostgresFindingStorage(db *sqlx.DB) *PostgresFindingStorage {
	return &PostgresFindingStorage{db: db}
}

func (s *PostgresFindingStorage) List() ([]recommendations.Finding, error) {
	findings := []recommendations.Finding{}

	err := s.db.Select(&findings, "SELECT * from findings")

	if err != nil {
		return nil, errors.Wrap(err, "postgres list findings")
	}

	return findings, err
}

func (s *PostgresFindingStorage) ListForStatus(status string) ([]recommendations.Finding, error) {
	f := []recommendations.Finding{}

	err := s.db.Select(&f, `SELECT id, identity_user as "identity.user", identity_role as "identity.role", identity_account as "identity.account", updated_at, event_count, status, document FROM findings WHERE status=$1`, status)
	if err != nil {
		return nil, errors.Wrap(err, "postgres list findings for status")
	}

	return f, nil
}

func (s *PostgresFindingStorage) Get(id string) (*recommendations.Finding, error) {
	var f recommendations.Finding

	err := s.db.Get(&f, `SELECT id, identity_user as "identity.user", identity_role as "identity.role", identity_account as "identity.account", updated_at, event_count, status, document FROM findings WHERE id=$1`, id)
	if err != nil {
		return nil, errors.Wrap(err, "postgres get finding")
	}

	return &f, nil
}

// FindByRole finds a matching finding by its role
func (s *PostgresFindingStorage) FindByRole(query FindByRoleQuery) (*recommendations.Finding, error) {
	var f recommendations.Finding

	err := s.db.Get(&f, `SELECT id, identity_user as "identity.user", identity_role as "identity.role", identity_account as "identity.account", updated_at, event_count, status, document FROM findings WHERE identity_role=$1 AND status=$2`, query.Role, query.Status)

	return &f, err
}

func (s *PostgresFindingStorage) CreateOrUpdate(f recommendations.Finding) error {
	_, err := s.db.Query("INSERT INTO findings (id, identity_user, identity_role, identity_account, updated_at, event_count, status, document) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		f.ID, f.Identity.User, f.Identity.Role, f.Identity.Account, f.UpdatedAt, f.EventCount, f.Status, f.Document,
	)
	return err
}

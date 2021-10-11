package storage

import (
	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type PostgresEventStorage struct {
	db *sqlx.DB
}

func NewPostgresEventStorage(db *sqlx.DB) *PostgresEventStorage {
	return &PostgresEventStorage{db: db}
}

func (s *PostgresEventStorage) ListForFinding(findingID string) ([]recommendations.AWSEvent, error) {
	e := []recommendations.AWSEvent{}

	err := s.db.Select(&e, `SELECT events.id, identity_user as "identity.user", identity_role as "identity.role", identity_account as "identity.account", events.time, data FROM events INNER JOIN actions ON actions.event_id = events.id WHERE actions.finding_id=$1`, findingID)

	if err != nil {
		return nil, errors.Wrap(err, "postgres list events")
	}

	return e, err
}

func (s *PostgresEventStorage) Create(e recommendations.AWSEvent) error {
	_, err := s.db.Query("INSERT INTO events (id, time, identity_user, identity_role, identity_account, data) VALUES ($1, $2, $3, $4, $5, $6)",
		e.ID, e.Time, e.Identity.User, e.Identity.Role, e.Identity.Account, e.Data,
	)
	return err
}

func (s *PostgresEventStorage) Get(id string) (*recommendations.AWSEvent, error) {
	var e recommendations.AWSEvent

	err := s.db.Get(&e, `SELECT events.id, identity_user as "identity.user", identity_role as "identity.role", identity_account as "identity.account", events.time, data FROM events WHERE id=$1`, id)

	if err != nil {
		return nil, errors.Wrap(err, "postgres get event")
	}

	return &e, err
}

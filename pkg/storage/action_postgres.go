package storage

import (
	"encoding/json"

	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type PostgresActionStorage struct {
	db *sqlx.DB
}

func NewPostgresActionStorage(db *sqlx.DB) *PostgresActionStorage {
	return &PostgresActionStorage{db: db}
}

func (s *PostgresActionStorage) List() ([]recommendations.AWSAction, error) {
	actions := []recommendations.AWSAction{}

	err := s.db.Select(&actions, "SELECT * from actions")

	if err != nil {
		return nil, errors.Wrap(err, "postgres list actions")
	}

	return actions, err
}

func (s *PostgresActionStorage) Get(id string) (*recommendations.AWSAction, error) {
	// @TODO add recommendations?

	var a DBAction
	err := s.db.Get(&a, `SELECT actions.id, finding_id, status, actions.time as "time", has_recommendations, enabled, events.id as "event.id", events.time as "event.time", events.identity_user as "event.identity.user", events.identity_role as "event.identity.role", events.identity_account as "event.identity.account", events.data as "eventData" FROM actions INNER JOIN events ON actions.event_id=events.id WHERE actions.id=$1 `, id)
	if err != nil {
		return nil, errors.Wrap(err, "postgres get action")
	}
	err = json.Unmarshal(a.EventData, &a.Event.Data)
	if err != nil {
		return nil, errors.Wrap(err, "postgres get action, unmarshalling event data")
	}
	return &a.AWSAction, nil

}

type DBAction struct {
	recommendations.AWSAction
	EventData []byte `db:"eventData"`
}

func (s *PostgresActionStorage) Add(a recommendations.AWSAction) error {
	data, err := json.Marshal(a.Event.Data)
	if err != nil {
		return errors.WithStack(err)
	}
	_, err = s.db.Query("INSERT INTO events (id, time, identity_user, identity_role, identity_account, data) VALUES ($1, $2, $3, $4, $5, $6)",
		a.Event.ID, a.Event.Time, a.Event.Identity.User, a.Event.Identity.Role, a.Event.Identity.Account, data,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = s.db.Query("INSERT INTO actions (id, finding_id, event_id, status, time, has_recommendations, enabled) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		a.ID, a.FindingID, a.Event.ID, a.Status, a.Time, a.HasRecommendations, a.Enabled,
	)
	return errors.WithStack(err)
}

func (s *PostgresActionStorage) ListForPolicy(findingID string) ([]recommendations.AWSAction, error) {
	actions := []DBAction{}

	err := s.db.Select(&actions, `SELECT actions.id, finding_id, status, actions.time as "time", has_recommendations, enabled, events.id as "event.id", events.time as "event.time", events.identity_user as "event.identity.user", events.identity_role as "event.identity.role", events.identity_account as "event.identity.account", events.data as "eventData" FROM actions INNER JOIN events ON actions.event_id=events.id  WHERE finding_id=$1`, findingID)

	if err != nil {
		return nil, errors.Wrap(err, "postgres list actions")
	}

	acts := []recommendations.AWSAction{}
	for _, a := range actions {
		err = json.Unmarshal(a.EventData, &a.Event.Data)
		if err != nil {
			return nil, errors.Wrap(err, "postgres get action, unmarshalling event data")
		}
		acts = append(acts, a.AWSAction)
	}
	return acts, err
}

func (s *PostgresActionStorage) ListEnabledActionsForFinding(findingID string) ([]recommendations.AWSAction, error) {
	actions := []DBAction{}

	err := s.db.Select(&actions, `SELECT actions.id, finding_id, status, actions.time as "time", has_recommendations, enabled, events.id as "event.id", events.time as "event.time", events.identity_user as "event.identity.user", events.identity_role as "event.identity.role", events.identity_account as "event.identity.account", events.data as "eventData" FROM actions INNER JOIN events ON actions.event_id=events.id  WHERE finding_id=$1 AND enabled = true`, findingID)

	if err != nil {
		return nil, errors.Wrap(err, "postgres list actions")
	}

	acts := []recommendations.AWSAction{}
	for _, a := range actions {
		err = json.Unmarshal(a.EventData, &a.Event.Data)
		if err != nil {
			return nil, errors.Wrap(err, "postgres get action, unmarshalling event data")
		}
		acts = append(acts, a.AWSAction)
	}
	return acts, err
}
func (s *PostgresActionStorage) SetStatus(id string, status string) error {
	_, err := s.db.Query("UPDATE actions SET status = $1 WHERE id = $2", status, id)
	if err != nil {
		return errors.Wrap(err, "postgres set status actions")
	}
	return nil
}

func (s *PostgresActionStorage) Update(action recommendations.AWSAction) error {
	_, err := s.db.Query("UPDATE actions SET finding_id=$2, status=$3, time=$4, has_recommendations=$5, enabled=$6 WHERE id = $1", action.ID, action.FindingID, action.Status, action.Time, action.HasRecommendations, action.Enabled)
	if err != nil {
		return errors.Wrap(err, "postgres update actions")
	}

	data, err := json.Marshal(action.Event.Data)
	if err != nil {
		return errors.Wrap(err, "postgres update actions, updating event")
	}
	_, err = s.db.Query("UPDATE events SET time=$2, identity_user=$3, identity_role=$4, identity_account=$5, data=$6  WHERE id = $1", action.Event.ID, action.Event.Time, action.Event.Identity.User, action.Event.Identity.Role, action.Event.Identity.Account, data)
	if err != nil {
		return errors.Wrap(err, "postgres update actions, updating event")
	}

	return nil
}

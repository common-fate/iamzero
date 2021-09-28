package tokens

import (
	"context"
	"database/sql"

	"github.com/common-fate/iamzero/pkg/crypto"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// PostgresDBTokenStorer is a token storage backend which uses Postgres
type PostgresDBTokenStorer struct {
	log    *zap.SugaredLogger
	tracer trace.Tracer
	db     *sqlx.DB
}

// NewPostgresDBTokenStorer initialises the AWS DynamoDB client and returns a new PostgresDBTokenStorer
func NewPostgresDBTokenStorer(ctx context.Context, db *sqlx.DB, log *zap.SugaredLogger, tracer trace.Tracer) (*PostgresDBTokenStorer, error) {
	return &PostgresDBTokenStorer{log, tracer, db}, nil
}

// Create a Token and store it in the database
func (s *PostgresDBTokenStorer) Create(ctx context.Context, name string) (*Token, error) {
	s.log.Info("creating token")

	ID, err := crypto.GenerateRandomToken()
	if err != nil {
		return nil, errors.Wrap(err, "generating token")
	}

	token := Token{
		ID:   ID,
		Name: name,
	}

	_, err = s.db.ExecContext(ctx, "INSERT INTO tokens (id, name) VALUES ($1, $2)", token.ID, token.Name)
	if err != nil {
		return nil, errors.Wrap(err, "inserting item")
	}

	return &token, nil
}

// Delete a token from the database
func (s *PostgresDBTokenStorer) Delete(ctx context.Context, id string) error {
	s.log.Info("deleting token")
	_, err := s.db.ExecContext(ctx, "DELETE FROM tokens WHERE id = $1", id)
	return err
}

// Get a token from the database
func (s *PostgresDBTokenStorer) Get(ctx context.Context, id string) (*Token, error) {

	var t Token
	err := s.db.GetContext(ctx, &t, "SELECT * FROM tokens WHERE id = $1", id)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "fetching item")
	}

	return &t, nil
}

// List all tokens
func (s *PostgresDBTokenStorer) List(ctx context.Context) ([]Token, error) {
	s.log.Info("listing tokens")
	ctx, span := s.tracer.Start(ctx, "DynamoDBTokenStorer.List")
	defer span.End()

	t := []Token{}
	err := s.db.SelectContext(ctx, &t, "SELECT * FROM tokens")
	if err != nil {
		return nil, err
	}

	return t, nil

}

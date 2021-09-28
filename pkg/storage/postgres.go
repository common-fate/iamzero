package storage

import (
	"flag"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

// PostgresStorage holds config for connecting to Postgres.
// It has an AddFlags method so it can be configured in the
// same way as other modules in the application.
type PostgresStorage struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string
}

func NewPostgresStorage() *PostgresStorage {
	return &PostgresStorage{}
}

func (p *PostgresStorage) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&p.Host, "postgres-host", "", "postgres host")
	fs.IntVar(&p.Port, "postgres-port", 5432, "postgres port")
	fs.StringVar(&p.Database, "postgres-db", "", "postgres database")
	fs.StringVar(&p.User, "postgres-user", "", "postgres user")
	fs.StringVar(&p.Password, "postgres-password", "", "postgres password")
	fs.StringVar(&p.SSLMode, "postgres-sslmode", "disable", "postgres SSL mode")
}

func (p *PostgresStorage) Connect() (*sqlx.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.Database, p.SSLMode)

	db, err := sqlx.Connect("postgres", psqlInfo)
	if err != nil {
		return nil, errors.Wrap(err, "connecting to postgres")
	}
	return db, nil
}

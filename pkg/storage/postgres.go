package storage

import (
	"flag"
	"fmt"

	_ "github.com/common-fate/iamzero/pkg/storage/migrations"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// PostgresStorage holds config for connecting to Postgres.
// It has an AddFlags method so it can be configured in the
// same way as other modules in the application.
type PostgresStorage struct {
	Host              string
	Port              int
	Database          string
	User              string
	Password          string
	SSLMode           string
	AutoRunMigrations bool
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
	fs.BoolVar(&p.AutoRunMigrations, "auto-run-migrations", true, "auto run migrations")
}

func (p *PostgresStorage) Connect(log *zap.SugaredLogger) (*sqlx.DB, error) {
	if p.AutoRunMigrations {
		log.Info("auto-migrate flag enabled, running postgres migrations")
		if err := p.RunMigration(); err != nil {
			return nil, err
		}
	}

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.Database, p.SSLMode)

	db, err := sqlx.Connect("postgres", psqlInfo)
	if err != nil {
		return nil, errors.Wrap(err, "connecting to postgres with these params : "+fmt.Sprintf("host=%s port=%d user=%s "+
			"password=**hidden** dbname=%s sslmode=%s",
			p.Host, p.Port, p.User, p.Database, p.SSLMode))
	}
	return db, nil
}

func (p *PostgresStorage) RunMigration() error {
	// We use the embed filesystem defined in _ "github.com/common-fate/iamzero/pkg/storage/migrations"
	// see this file for full details
	m, err := migrate.New(
		"embed://",
		p.ConnectionString())
	if err != nil {
		return errors.Wrap(err, "error connecting to database while running migrations")
	}
	err = m.Up()
	if err != migrate.ErrNoChange {
		return errors.Wrap(err, "applying migrations")
	}
	return nil
}

func (p *PostgresStorage) PsqlInfoString() string {
	return fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.Database, p.SSLMode)
}
func (p *PostgresStorage) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		p.User, p.Password, p.Host, p.Port, p.Database, p.SSLMode)
}

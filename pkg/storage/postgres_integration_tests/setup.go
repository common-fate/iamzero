// +build postgres

package postgresintegrationtests

import (
	"crypto/rand"
	"fmt"
	"math/big"

	_ "github.com/common-fate/iamzero/pkg/storage/migrations"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

var database *sqlx.DB
var dbName string

func init() {
	_, err := GetDB()
	if err != nil {
		panic(err)
	}
}

// generateRandomDatabaseName gives us a short 10 char random database string
// which we can use as part of a temporary testing database name
func generateRandomDatabaseName() (string, error) {
	const n = 10
	const letters = "0123456789abcdefghijklmnopqrstuvwxyz"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		ret[i] = letters[num.Int64()]
	}

	return string(ret), nil
}

// GetDB connects to a localhost database and provisions a temporary testing database,
// so that in development our unit test data doesn't pollute the main database we are working from.
func GetDB() (*sqlx.DB, error) {
	var err error
	if database != nil {
		return database, nil
	}
	var initialDB *sqlx.DB

	initialDB, err = sqlx.Connect("postgres", "host=localhost port=5432 user=postgres "+
		"password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		return nil, err
	}

	name, err := generateRandomDatabaseName()
	if err != nil {
		return nil, err
	}
	dbName = "test_" + name

	_, err = initialDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		return nil, err
	}
	initialDB.Close()

	fmt.Printf("created db: %s\n", dbName)

	psqlString := fmt.Sprintf("host=localhost port=5432 user=postgres "+
		"password=postgres dbname=%s sslmode=disable", dbName)

	database, err = sqlx.Connect("postgres", psqlString)
	if err != nil {
		return nil, err
	}

	driver, err := postgres.WithInstance(database.DB, &postgres.Config{})
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://../migrations",
		"postgres", driver)

	if err != nil {
		return nil, errors.Wrap(err, "error connecting to database while running migrations")
	}
	err = m.Up()
	if err != migrate.ErrNoChange {
		return nil, errors.Wrap(err, "applying migrations")
	}

	return database, nil
}

// CloseDB tries to drop the testing database and then close the connection
func CloseDB() {
	if database != nil {
		_, err := database.Exec(fmt.Sprintf("DROP DATABASE %s", dbName))
		if err != nil {
			fmt.Printf("Error while dropping database %s: %s. The database may not have been deleted.", dbName, err)
		}
		database.Close()
	}
}

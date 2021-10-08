package migrations

import (
	"fmt"

	"embed"
	"net/http"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	_ "github.com/lib/pq"
)

/*
This file sets up an embedded filesystem that can be used be golang-migrate as a source for migration files, this way, the migrations should be bundled up into the binary
and dont need to be added as a seperate file/folder in docker containers.

to use this, add this import line to the file with the migrator code.
the effect of this import is that the init() function defined in this file will run on first load setting up the embeded filesystem as the "embed://" source
see this example for reference https://github.com/golang-migrate/migrate/issues/514#issuecomment-789794159
import ( _ "github.com/common-fate/iamzero/pkg/storage/migrations" )
*/

// Go:embed cant cross module boundaries when looking for files in this path, therefor the migrations folder has to be within this module
//go:embed *.sql
var static embed.FS

// init runs when the package in imported
func init() {
	source.Register("embed", &driver{})
}

type driver struct {
	httpfs.PartialDriver
}

func (d *driver) Open(rawURL string) (source.Driver, error) {
	err := d.PartialDriver.Init(http.FS(static), ".")
	if err != nil {

		fmt.Printf("err: %v\n", err)
		return nil, err
	}

	return d, nil
}

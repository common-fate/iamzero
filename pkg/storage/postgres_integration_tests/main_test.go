// +build postgres

package postgresintegrationtests

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	defer CloseDB()
	os.Exit(m.Run())
}

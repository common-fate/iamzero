// +build postgres

package postgresintegrationtests

import (
	"testing"

	"github.com/common-fate/iamzero/pkg/tokens"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// an example test to verify that postgres testing works as expected.
func Test_BasicInsert(t *testing.T) {
	db, err := GetDB()
	if err != nil {
		t.Fatal(err)
	}

	id := uuid.New()

	_, err = db.Query("INSERT INTO tokens (id, name) VALUES ($1, $2)", id, "test")
	if err != nil {
		t.Fatal(err)
	}

	var tok tokens.Token

	err = db.Get(&tok, "SELECT * from tokens LIMIT 1")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, id.String(), tok.ID)
	assert.Equal(t, "test", tok.Name)
}

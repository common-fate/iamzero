// +build postgres

package postgresintegrationtests

import (
	"testing"

	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/common-fate/iamzero/pkg/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func mockEvent() recommendations.AWSEvent {
	return recommendations.AWSEvent{
		ID: uuid.NewString(),
		Identity: recommendations.AWSIdentity{
			User:    "testUser",
			Role:    "testRole",
			Account: "123456789012",
		},
		Data: recommendations.AWSData{},
		Time: "2021-09-02T04:29:14Z",
	}
}

func Test_GetEvent(t *testing.T) {
	db, err := GetDB()
	if err != nil {
		t.Fatal(err)
	}

	s := storage.NewPostgresEventStorage(db)

	_, err = s.ListForFinding(uuid.NewString())
	assert.NoError(t, err)
}

func Test_CreateEvent(t *testing.T) {
	db, err := GetDB()
	if err != nil {
		t.Fatal(err)
	}

	e := mockEvent()

	s := storage.NewPostgresEventStorage(db)

	err = s.Create(e)
	assert.NoError(t, err)
}

func Test_CreateAndGetEvent(t *testing.T) {
	db, err := GetDB()
	if err != nil {
		t.Fatal(err)
	}

	e := mockEvent()

	s := storage.NewPostgresEventStorage(db)

	err = s.Create(e)
	if err != nil {
		t.Fatal(err)
	}

	result, err := s.Get(e.ID)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, e, *result)
}

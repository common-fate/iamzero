// +build postgres

package postgresintegrationtests

import (
	"testing"
	"time"

	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/common-fate/iamzero/pkg/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func mockAWSAction(findingID string) recommendations.AWSAction {
	return recommendations.AWSAction{
		ID: uuid.NewString(),
		Event: recommendations.AWSEvent{
			ID: uuid.NewString(),
			Identity: recommendations.AWSIdentity{

				User:    "testUser",
				Role:    "testRole",
				Account: "123456789012",
			}, Data: recommendations.AWSData{}, Time: "2021-09-02T04:29:14Z"},
		Status:             "test",
		FindingID:          findingID,
		Time:               time.Now(),
		HasRecommendations: true,
		Enabled:            true, SelectedLeastPrivilegePolicyID: "",
	}
}

func Test_CreateAction(t *testing.T) {
	db, err := GetDB()
	if err != nil {
		t.Fatal(err)
	}

	s := storage.NewPostgresFindingStorage(db)

	finding := mockFinding()

	err = s.CreateOrUpdate(finding)
	if err != nil {
		t.Fatal(err)
	}
	as := storage.NewPostgresActionStorage(db)
	action := mockAWSAction(finding.ID)
	err = as.Add(action)
	if err != nil {
		t.Fatal(err)
	}

}

func Test_GetAction(t *testing.T) {
	db, err := GetDB()
	if err != nil {
		t.Fatal(err)
	}

	s := storage.NewPostgresFindingStorage(db)

	finding := mockFinding()

	err = s.CreateOrUpdate(finding)
	if err != nil {
		t.Fatal(err)
	}
	as := storage.NewPostgresActionStorage(db)
	action := mockAWSAction(finding.ID)
	err = as.Add(action)
	if err != nil {
		t.Fatal(err)
	}

	a, err := as.Get(action.ID)
	if err != nil {
		t.Fatal(err)
	}
	a.Time = action.Time
	assert.Equal(t, action, *a)
}

func Test_ListForPolicy(t *testing.T) {
	db, err := GetDB()
	if err != nil {
		t.Fatal(err)
	}

	db.Query("DELETE FROM actions")
	db.Query("DELETE FROM events")
	db.Query("DELETE FROM findings")

	s := storage.NewPostgresFindingStorage(db)

	finding := mockFinding()

	err = s.CreateOrUpdate(finding)
	if err != nil {
		t.Fatal(err)
	}
	as := storage.NewPostgresActionStorage(db)
	action := mockAWSAction(finding.ID)
	err = as.Add(action)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := as.ListForPolicy(finding.ID)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, actual, 1)

	// ignore the timestamp for now
	actual[0].Time = action.Time

	assert.Equal(t, action, actual[0])
}

func Test_ListEnabledActionsForFinding(t *testing.T) {
	db, err := GetDB()
	if err != nil {
		t.Fatal(err)
	}

	db.Query("DELETE FROM actions")
	db.Query("DELETE FROM events")
	db.Query("DELETE FROM findings")

	s := storage.NewPostgresFindingStorage(db)

	finding := mockFinding()

	err = s.CreateOrUpdate(finding)
	if err != nil {
		t.Fatal(err)
	}
	as := storage.NewPostgresActionStorage(db)
	action := mockAWSAction(finding.ID)
	err = as.Add(action)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := as.ListEnabledActionsForFinding(finding.ID)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, actual, 1)

	// ignore the timestamp for now
	actual[0].Time = action.Time

	assert.Equal(t, action, actual[0])
}

func Test_SetStatus(t *testing.T) {
	db, err := GetDB()
	if err != nil {
		t.Fatal(err)
	}

	s := storage.NewPostgresFindingStorage(db)

	finding := mockFinding()

	err = s.CreateOrUpdate(finding)
	if err != nil {
		t.Fatal(err)
	}
	as := storage.NewPostgresActionStorage(db)
	action := mockAWSAction(finding.ID)
	err = as.Add(action)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := as.Get(action.ID)

	assert.Equal(t, action.Status, a.Status)

	as.SetStatus(action.ID, "test_status")

	a, _ = as.Get(action.ID)

	assert.Equal(t, "test_status", a.Status)
}

func Test_Update(t *testing.T) {
	db, err := GetDB()
	if err != nil {
		t.Fatal(err)
	}

	s := storage.NewPostgresFindingStorage(db)

	finding := mockFinding()

	err = s.CreateOrUpdate(finding)
	if err != nil {
		t.Fatal(err)
	}
	as := storage.NewPostgresActionStorage(db)
	action := mockAWSAction(finding.ID)
	err = as.Add(action)
	if err != nil {
		t.Fatal(err)
	}

	action.Status = "test123"
	err = as.Update(action)
	if err != nil {
		t.Fatal(err)
	}
	a, err := as.Get(action.ID)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "test123", a.Status)
}

// +build postgres

package postgresintegrationtests

import (
	"database/sql"
	"testing"
	"time"

	"github.com/common-fate/iamzero/pkg/policies"
	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/common-fate/iamzero/pkg/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func mockFinding() recommendations.Finding {
	return recommendations.Finding{
		ID: uuid.NewString(),
		Identity: recommendations.ProcessedAWSIdentity{
			User:    "testUser",
			Role:    "testRole",
			Account: "123456789012",
		},
		UpdatedAt:  time.Now(),
		EventCount: 10,
		Status:     "test",
		Document: policies.AWSIAMPolicy{
			Version: "2012-10-17",
			Statement: []policies.AWSIAMStatement{
				{
					Sid:      "1",
					Effect:   "Allow",
					Action:   []string{"sts:AssumeRole"},
					Resource: []string{"arn:aws:iam::111222333444:role/target"},
				},
			},
		},
	}
}

func Test_CreateFinding(t *testing.T) {
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
}

func Test_GetFinding(t *testing.T) {
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

	actual, err := s.Get(finding.ID)
	if err != nil {
		t.Fatal(err)
	}
	// ignore the timestamp for now
	actual.UpdatedAt = finding.UpdatedAt

	assert.Equal(t, finding, *actual)
}

func Test_ListForStatus(t *testing.T) {
	db, err := GetDB()
	if err != nil {
		t.Fatal(err)
	}

	db.Query("DELETE FROM findings")

	s := storage.NewPostgresFindingStorage(db)

	finding := mockFinding()

	err = s.CreateOrUpdate(finding)
	if err != nil {
		t.Fatal(err)
	}

	actual, err := s.ListForStatus(finding.Status)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, actual, 1)

	// ignore the timestamp for now
	actual[0].UpdatedAt = finding.UpdatedAt

	assert.Equal(t, finding, actual[0])
}

func Test_FindByRole(t *testing.T) {
	db, err := GetDB()
	if err != nil {
		t.Fatal(err)
	}

	db.Query("DELETE FROM findings")

	s := storage.NewPostgresFindingStorage(db)

	finding := mockFinding()

	err = s.CreateOrUpdate(finding)
	if err != nil {
		t.Fatal(err)
	}

	actual, err := s.FindByRole(storage.FindByRoleQuery{
		Role:   finding.Identity.Role,
		Status: finding.Status,
	})
	if err != nil {
		t.Fatal(err)
	}
	// ignore the timestamp for now
	actual.UpdatedAt = finding.UpdatedAt

	assert.Equal(t, finding, *actual)
}

func Test_FindByRoleNotFound(t *testing.T) {
	// if the role doesn't exist, we should return sql.ErrNoRows
	db, err := GetDB()
	if err != nil {
		t.Fatal(err)
	}

	s := storage.NewPostgresFindingStorage(db)

	_, err = s.FindByRole(storage.FindByRoleQuery{
		Role:   "notfound",
		Status: "notfound",
	})
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

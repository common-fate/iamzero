package recommendations_test

import (
	"testing"

	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/stretchr/testify/assert"
)

func TestGetRoleOrUserNameFromARN(t *testing.T) {
	// +=,.@-_ are all valid characters which can be included in an IAM role
	arn := "arn:aws:sts::123456789012:assumed-role/iamzero-test-role+=,.@-_/iamzero-test"
	role, err := recommendations.GetRoleOrUserNameFromARN(arn)
	if err != nil {
		t.Fatal(err)
	}
	if role != "iamzero-test-role+=,.@-_" {
		t.Errorf("role did not match, received: %s", role)
	}
}

func TestGetRoleOrUserNameFromARNWithUser(t *testing.T) {
	arn_user := "arn:aws:iam::123456789012:user/iamzero-test-user+=,.@-_"
	username, err := recommendations.GetRoleOrUserNameFromARN(arn_user)
	if err != nil {
		t.Fatal(err)
	}
	if username != "iamzero-test-user+=,.@-_" {
		t.Errorf("username did not match, received: %s", username)
	}
}

func TestGetRoleOrUserNameFromARNWithInvalidArn(t *testing.T) {
	invalid_arn := "arn:aws:iam::123456789012:assumed-role/iamzero-test-user+=,.@-_"
	_, err := recommendations.GetRoleOrUserNameFromARN(invalid_arn)
	if err == nil {
		t.Errorf("expected error, got nil for arn %s", invalid_arn)
	}
}

func TestGetRoleOrUserNameFromARNWithDemoRole(t *testing.T) {
	arn := "arn:aws:iam::123456789012:role/iamzero-test-role"
	_, err := recommendations.GetRoleOrUserNameFromARN(arn)
	if err != nil {
		t.Fatal(err)
	}
}

func ParseRealRoleARN_ReplacesAssumedRoleWithActualRole(t *testing.T) {
	arn := "arn:aws:sts::123456789012:assumed-role/iamzero-test-role/iamzero-test"
	role, err := recommendations.ExtractRoleARNFromSession(arn)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "arn:aws:iam::123456789012:role/iamzero-test-role", role)
}

func ParseRealRoleARN_ReturnsNilIfAssumedRoleIsNotGiven(t *testing.T) {
	arn := "arn:aws:iam::123456789012:role/iamzero-test-role"
	role, err := recommendations.ExtractRoleARNFromSession(arn)
	if err != nil {
		t.Fatal(err)
	}

	assert.Nil(t, role)
}

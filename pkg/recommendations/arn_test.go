package recommendations_test

import (
	"testing"

	"github.com/common-fate/iamzero/pkg/recommendations"
)

func TestGetRoleOrUserNameFromARN(t *testing.T) {
	// +=,.@-_ are all valid characters which can be included in an IAM role
	arn := "arn:aws:sts::123456789:assumed-role/iamzero-test-role+=,.@-_/iamzero-test"
	role, err := recommendations.GetRoleOrUserNameFromARN(arn)
	if err != nil {
		t.Fatal(err)
	}
	if role != "iamzero-test-role+=,.@-_" {
		t.Errorf("role did not match, received: %s", role)
	}
	arn_user := "arn:aws:iam::123456789012:user/iamzero-test-user+=,.@-_"
	username, err := recommendations.GetRoleOrUserNameFromARN(arn_user)
	if err != nil {
		t.Fatal(err)
	}
	if username != "iamzero-test-user+=,.@-_" {
		t.Errorf("username did not match, received: %s", role)
	}

	invalid_arn := "arn:aws:iam::123456789012:assumed-role/iamzero-test-user+=,.@-_"
	_, err = recommendations.GetRoleOrUserNameFromARN(invalid_arn)
	if err == nil {
		t.Errorf("expected error, got nil for arn %s", invalid_arn)
	}
}

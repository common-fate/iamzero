package recommendations_test

import (
	"testing"

	"github.com/common-fate/iamzero/pkg/recommendations"
)

func TestGetRoleNameFromARN(t *testing.T) {
	// +=,.@-_ are all valid characters which can be included in an IAM role
	arn := "arn:aws:sts::123456789:assumed-role/iamzero-test-role+=,.@-_/iamzero-test"
	role, err := recommendations.GetRoleNameFromARN(arn)
	if err != nil {
		t.Fatal(err)
	}
	if role != "iamzero-test-role+=,.@-_" {
		t.Errorf("role did not match, received: %s", role)
	}
}

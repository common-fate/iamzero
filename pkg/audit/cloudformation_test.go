package audit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseIDFromCDKPath_works(t *testing.T) {
	path := "CdkExampleStack/iamzero-example-role/Resource"
	result := parseIDFromCDKPath(path)

	assert.Equal(t, "iamzero-example-role", result)
}

package recommendations

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalIAMStatements_Many(t *testing.T) {
	bytes := []byte(`
	[
		{
			"Sid": "1",
			"Effect": "Allow",
			"Action": "s3:GetObject",
			"Resource": "*"
		}
	]
	`)
	var s IAMStatements
	err := json.Unmarshal(bytes, &s)
	if err != nil {
		t.Fatal(err)
	}
}

// Some AWS IAM statements come back as a single object,
// i.e. not in an array! Our JSON unmarshaller needs to
// handle this case.
func TestUnmarshalIAMStatements_Single(t *testing.T) {
	// add some initial whitespace too, to make sure we can handle that.
	bytes := []byte(`


		{
			"Sid": "1",
			"Effect": "Allow",
			"Action": "s3:GetObject",
			"Resource": "*"
		}
	`)
	var s IAMStatements
	err := json.Unmarshal(bytes, &s)
	if err != nil {
		t.Fatal(err)
	}
}

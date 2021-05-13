package recommendations

import (
	"fmt"
	"regexp"
)

// GetRoleNameFromARN parses an ARN to return a role name,
// or returns an error if the ARN is malformed.
func GetRoleNameFromARN(arn string) (string, error) {
	re, err := regexp.Compile(`arn:aws:sts::[\d]+:assumed-role/([\w\d+=,.@_-]+)`)
	if err != nil {
		return "", err
	}
	role := re.FindStringSubmatch(arn)
	if role == nil {
		return "", fmt.Errorf("could not find role in ARN %s", arn)
	}
	return role[1], nil
}

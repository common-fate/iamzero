package recommendations

import (
	"fmt"
	"regexp"
)

// GetRoleOrUserNameFromARN parses an ARN to return a role name, or a user name if it is an IAM user
// or returns an error if the ARN is malformed.
func GetRoleOrUserNameFromARN(arn string) (string, error) {
	re, err := regexp.Compile(`arn:aws:(?:sts::[\d]+:assumed-role|iam::\d{12}:user)/([\w\d+=,.@_-]+)`)
	if err != nil {
		return "", err
	}
	role := re.FindStringSubmatch(arn)
	if role == nil {
		return "", fmt.Errorf("could not find role in ARN %s", arn)
	}
	return role[1], nil
}

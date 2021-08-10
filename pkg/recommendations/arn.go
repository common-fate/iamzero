package recommendations

import (
	"fmt"
	"regexp"
)

// GetRoleOrUserNameFromARN parses an ARN to return a role name, or a user name if it is an IAM user
// or returns an error if the ARN is malformed.
//
// The function replaces assumed role ARNs with the actual role ARN.
func GetRoleOrUserNameFromARN(arn string) (string, error) {
	re, err := regexp.Compile(`arn:aws:(?:sts::[\d]+:assumed-role|iam::\d{12}:user|iam::\d{12}:role)/([\w\d+=,.@_-]+)`)
	if err != nil {
		return "", err
	}
	role := re.FindStringSubmatch(arn)
	if role == nil {
		return "", fmt.Errorf("could not find role in ARN %s", arn)
	}
	return role[1], nil
}

// ParseRealRoleARN parses a session role ARN and returns the IAM role ARN
// If the provided ARN is not a session role ARN, nil is returned
// See https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_identifiers.html
func ExtractRoleARNFromSession(arn string) (*string, error) {
	re, err := regexp.Compile(`arn:aws:sts::(\d{12}):assumed-role/([\w\d+=,.@_-]+)/[\w\d+=,.@_-]+`)
	if err != nil {
		return nil, err
	}
	matches := re.FindStringSubmatch(arn)
	if matches != nil {
		awsAccount := matches[1]
		roleName := matches[2]
		result := fmt.Sprintf("arn:aws:iam::%s:role/%s", awsAccount, roleName)
		return &result, nil
	}
	return nil, nil
}

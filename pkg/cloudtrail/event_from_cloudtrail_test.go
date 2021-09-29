package cloudtrail

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/stretchr/testify/assert"
)

func getTestLogEntry() CloudTrailLogEntry {
	return CloudTrailLogEntry{
		UserIdentity: CloudTrailUserIdentity{
			Type:          aws.String("AssumedRole"),
			PrincipalID:   aws.String("AROAUAMTP2WEJUZJXFJX7:test-role"),
			ARN:           aws.String("arn:aws:sts::123456789012:assumed-role/CdkExampleStack-iamzerooverprivilegedrole3B0B7D55-1TIJOTM9XXJZ7/test-role"),
			AccountID:     aws.String("123456789012"),
			SessionIssuer: aws.String("{type=Role, principalid=AROAUAMTP2WEJUZJXFJX7, arn=arn:aws:iam::123456789012:role/CdkExampleStack-iamzerooverprivilegedrole3B0B7D55-1TIJOTM9XXJZ7, accountid=123456789012, username=CdkExampleStack-iamzerooverprivilegedrole3B0B7D55-1TIJOTM9XXJZ7}"),
		},
		EventTime:         aws.String("2021-09-02T04:29:14Z"),
		EventSource:       aws.String("s3.amazonaws.com"),
		EventName:         aws.String("HeadObject"),
		RequestParameters: aws.String("{\"bucketName\":\"testbucket\",\"Host\":\"testbucket.s3.ap-southeast-2.amazonaws.com\",\"key\":\"README.md\"}"),
	}
}

func TestTryConvertToEvent_Works(t *testing.T) {
	e := getTestLogEntry()

	result, err := e.TryConvertToEvent()
	assert.NoError(t, err)

	expected := &recommendations.AWSEvent{
		Time: "2021-09-02T04:29:14Z",
		Identity: recommendations.AWSIdentity{
			User:    "AROAUAMTP2WEJUZJXFJX7:test-role",
			Role:    "arn:aws:sts::123456789012:assumed-role/CdkExampleStack-iamzerooverprivilegedrole3B0B7D55-1TIJOTM9XXJZ7/test-role",
			Account: "123456789012",
		},
		Data: recommendations.AWSData{
			Type:      "awsAction",
			Service:   "s3",
			Operation: "HeadObject",
			Parameters: map[string]interface{}{
				"Bucket": "testbucket",
			},
		},
	}
	assert.Equal(t, expected, result)
}

func TestTryConvertToEvent_ReturnsErrorIfNoMapping(t *testing.T) {
	e := CloudTrailLogEntry{
		UserIdentity: CloudTrailUserIdentity{
			Type:          aws.String("AssumedRole"),
			PrincipalID:   aws.String("AROAUAMTP2WEJUZJXFJX7:test-role"),
			ARN:           aws.String("arn:aws:sts::123456789012:assumed-role/CdkExampleStack-iamzerooverprivilegedrole3B0B7D55-1TIJOTM9XXJZ7/test-role"),
			AccountID:     aws.String("123456789012"),
			SessionIssuer: aws.String("{type=Role, principalid=AROAUAMTP2WEJUZJXFJX7, arn=arn:aws:iam::123456789012:role/CdkExampleStack-iamzerooverprivilegedrole3B0B7D55-1TIJOTM9XXJZ7, accountid=123456789012, username=CdkExampleStack-iamzerooverprivilegedrole3B0B7D55-1TIJOTM9XXJZ7}"),
		},
		EventTime: aws.String("2021-09-02T04:29:14Z"),
		EventName: aws.String("NonExistentAPICall"),
	}

	_, err := e.TryConvertToEvent()
	assert.ErrorIs(t, err, ErrNoMapping)
}

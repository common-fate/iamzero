package cloudtrail

import (
	"testing"

	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAggregatorRead_Works(t *testing.T) {
	e := getTestLogEntry()
	agg := NewAggregator()

	err := agg.Read(e)
	assert.NoError(t, err)

	res := agg.GetEvents()

	expected := []recommendations.AWSEvent{

		{
			ID:   uuid.NewString(),
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
		},
	}
	assert.Equal(t, expected, res)
}

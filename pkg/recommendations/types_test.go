package recommendations

import "testing"

func TestHashEvent_Works(t *testing.T) {
	e := AWSEvent{
		Time: "2021-09-02T04:29:14Z",
		Identity: AWSIdentity{
			User:    "AROAUAMTP2WEJUZJXFJX7:test-role",
			Role:    "arn:aws:sts::123456789012:assumed-role/CdkExampleStack-iamzerooverprivilegedrole3B0B7D55-1TIJOTM9XXJZ7/test-role",
			Account: "123456789012",
		},
		Data: AWSData{
			Type:      "awsAction",
			Service:   "s3",
			Operation: "HeadObject",
			Parameters: map[string]interface{}{
				"Bucket": "testbucket",
			},
		},
	}

	_, err := HashEvent(e)
	if err != nil {
		t.Fatal(err)
	}
}

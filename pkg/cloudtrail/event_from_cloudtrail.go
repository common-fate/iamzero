package cloudtrail

import (
	"encoding/json"

	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/pkg/errors"
)

var ErrNoMapping = errors.New("could not convert CloudTrail entry to IAM Zero event")

// safeStringEquals checks if the first string pointer
// is nil to avoid a nil dereference error
func safeStringEquals(a *string, b string) bool {
	return a != nil && *a == b
}

// TryConvertToEvent tries to map the CloudTrail log entry
// to an AWSEvent. If the log entry cannot be processed, ErrNoMapping is returned.
func (c *CloudTrailLogEntry) TryConvertToEvent() (*recommendations.AWSEvent, error) {
	if safeStringEquals(c.EventSource, "s3.amazonaws.com") {
		// S3
		if safeStringEquals(c.EventName, "HeadObject") {
			var params map[string]string
			err := json.Unmarshal([]byte(*c.RequestParameters), &params)
			if err != nil {
				return nil, errors.Wrap(err, "unmarshalling params")
			}

			event := recommendations.AWSEvent{
				Time: *c.EventTime,
				Identity: recommendations.AWSIdentity{
					User:    *c.UserIdentity.PrincipalID,
					Role:    *c.UserIdentity.ARN,
					Account: *c.UserIdentity.AccountID,
				},
				Data: recommendations.AWSData{
					Type:      "awsAction",
					Service:   "s3",
					Operation: "HeadObject",
					Parameters: map[string]interface{}{
						"Bucket": params["bucketName"],
					},
				},
			}
			return &event, nil
		}
		if safeStringEquals(c.EventName, "ListObjects") {
			var params map[string]string
			err := json.Unmarshal([]byte(*c.RequestParameters), &params)
			if err != nil {
				return nil, errors.Wrap(err, "unmarshalling params")
			}

			event := recommendations.AWSEvent{
				Time: *c.EventTime,
				Identity: recommendations.AWSIdentity{
					User:    *c.UserIdentity.PrincipalID,
					Role:    *c.UserIdentity.ARN,
					Account: *c.UserIdentity.AccountID,
				},
				Data: recommendations.AWSData{
					Type:      "awsAction",
					Service:   "s3",
					Operation: "ListObjects",
					Parameters: map[string]interface{}{
						"Bucket": params["bucketName"],
					},
				},
			}
			return &event, nil
		}
	}
	// we weren't able to convert the log entry into an event
	return nil, ErrNoMapping
}

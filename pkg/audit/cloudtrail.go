package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/athena"
	"github.com/aws/aws-sdk-go-v2/service/athena/types"
	"go.uber.org/zap"
)

// CloudTrailAuditor queries CloudTrail logs
// via Athena
type CloudTrailAuditor struct {
	log                    *zap.SugaredLogger
	athenaCloudTrailBucket string
	athenaOutputLocation   string
}

type CloudTrailAuditorParams struct {
	Log                    *zap.SugaredLogger
	AthenaCloudTrailBucket string
	AthenaOutputLocation   string
}

func NewCloudTrailAuditor(params *CloudTrailAuditorParams) *CloudTrailAuditor {
	return &CloudTrailAuditor{
		log:                    params.Log,
		athenaCloudTrailBucket: params.AthenaCloudTrailBucket,
		athenaOutputLocation:   params.AthenaOutputLocation,
	}
}

// CloudTrailLogEntry is an invidual audit log stored in CloudTrail
// which we query with Athena
type CloudTrailLogEntry struct {
	UserIdentity      *string
	EventTime         *string
	EventSource       *string
	EventName         *string
	ErrorCode         *string
	ErrorMessage      *string
	RequestParameters *string
}

// CloudTrailAuditor queries CloudTrail logs
// via Athena to find AWS actions relating to
// a particular IAM role
func (a *CloudTrailAuditor) GetActionsForRole(ctx context.Context, account, role string) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}
	svc := athena.NewFromConfig(cfg)

	assumedRoleARN := fmt.Sprintf("arn:aws:sts::%s:assumed-role/%s/%%", account, role)
	roleARN := fmt.Sprintf("arn:aws:iam::%s:role/%s", account, role)

	query := fmt.Sprintf(`SELECT useridentity, eventtime, eventsource, eventname, errorcode, errormessage, requestparameters
	FROM %s
	WHERE useridentity.arn LIKE '%s'
			OR useridentity.arn = '%s'
`, a.athenaCloudTrailBucket, assumedRoleARN, roleARN)

	out, err := svc.StartQueryExecution(ctx, &athena.StartQueryExecutionInput{
		QueryString: &query,
		ResultConfiguration: &types.ResultConfiguration{
			OutputLocation: &a.athenaOutputLocation,
		},
		WorkGroup: aws.String("primary"),
	})
	if err != nil {
		return err
	}

	finished := false

	for !finished {
		fmt.Printf("waiting for Athena query... query-id=%s\n", *out.QueryExecutionId)
		q, err := svc.GetQueryExecution(ctx, &athena.GetQueryExecutionInput{
			QueryExecutionId: aws.String(*out.QueryExecutionId),
		})
		if err != nil {
			return err
		}

		state := q.QueryExecution.Status.State
		if state == types.QueryExecutionStateRunning || state == types.QueryExecutionStateQueued {
			// wait a second before trying again
			time.Sleep(time.Second * 1)
		} else if state == types.QueryExecutionStateSucceeded {
			finished = true
		} else {
			reason := *q.QueryExecution.Status.StateChangeReason
			return fmt.Errorf("error while querying athena: %s", reason)
		}
	}
	res, err := svc.GetQueryResults(ctx, &athena.GetQueryResultsInput{
		QueryExecutionId: out.QueryExecutionId,
	})
	if err != nil {
		return err
	}

	entries := []CloudTrailLogEntry{}
	for i, r := range res.ResultSet.Rows {
		if i == 0 {
			continue
		}
		entry := CloudTrailLogEntry{
			UserIdentity:      r.Data[0].VarCharValue,
			EventTime:         r.Data[1].VarCharValue,
			EventSource:       r.Data[2].VarCharValue,
			EventName:         r.Data[3].VarCharValue,
			ErrorCode:         r.Data[4].VarCharValue,
			ErrorMessage:      r.Data[5].VarCharValue,
			RequestParameters: r.Data[6].VarCharValue,
		}
		entries = append(entries, entry)

	}

	a.log.With("entries", entries).Info("found entries")

	return nil
}

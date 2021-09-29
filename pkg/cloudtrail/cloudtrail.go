package cloudtrail

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

type CloudTrailUserIdentity struct {
	Type          *string
	PrincipalID   *string
	ARN           *string
	AccountID     *string
	InvokedBy     *string
	SessionIssuer *string
}

// CloudTrailLogEntry is an invidual audit log stored in CloudTrail
// which we query with Athena
type CloudTrailLogEntry struct {
	UserIdentity      CloudTrailUserIdentity
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

	query := fmt.Sprintf(`SELECT useridentity.type, useridentity.principalid, useridentity.arn, useridentity.accountid, useridentity.invokedby, useridentity.sessioncontext.sessionissuer, eventtime, eventsource, eventname, errorcode, errormessage, requestparameters
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

	agg := NewAggregator()

	for i, r := range res.ResultSet.Rows {
		if i == 0 {
			continue
		}
		entry := CloudTrailLogEntry{
			UserIdentity: CloudTrailUserIdentity{
				Type:          r.Data[0].VarCharValue,
				PrincipalID:   r.Data[1].VarCharValue,
				ARN:           r.Data[2].VarCharValue,
				AccountID:     r.Data[3].VarCharValue,
				InvokedBy:     r.Data[4].VarCharValue,
				SessionIssuer: r.Data[5].VarCharValue,
			},
			EventTime:         r.Data[6].VarCharValue,
			EventSource:       r.Data[7].VarCharValue,
			EventName:         r.Data[8].VarCharValue,
			ErrorCode:         r.Data[9].VarCharValue,
			ErrorMessage:      r.Data[10].VarCharValue,
			RequestParameters: r.Data[11].VarCharValue,
		}

		err := agg.Read(entry)
		if err != nil {
			return err
		}

	}

	events := agg.GetEvents()

	a.log.With("events", events).Info("found events")

	return nil
}

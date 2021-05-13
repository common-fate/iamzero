package recommendations

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/common-fate/iamzero/pkg/events"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type KMSRecommendation struct {
	ID        string
	AccountID string
	KeyARN    string
	RoleARN   string
	Comment   string
}

// This recommendation is made when a DynamoDB table is encrypted with a
// customer-managed KMS key (CMK), but the role accessing the database
// does not have permission to use the key.
//
// If it exists, a recommendation is created to add a grant to the KMS key
// itself to allow the role to access it, as well as updating the IAM policy
// associated with the role to allow it to access the key.
func dynamoDBkmsRecommendation(e events.AWSEvent) (Advice, error) {
	ctx := context.TODO()
	if e.Data.Service == "dynamodb" &&
		e.Data.Operation == "GetItem" &&
		e.Data.ExceptionCode == "AccessDeniedException" &&
		strings.Contains(e.Data.ExceptionMessage, "KMS key access denied error") {

		// get the dynamoDB table name from the alert
		table := e.Data.Parameters["TableName"].(string)

		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, err
		}

		ddb := dynamodb.NewFromConfig(cfg)

		tableResult, err := ddb.DescribeTable(ctx, &dynamodb.DescribeTableInput{
			TableName: &table,
		})
		if err != nil {
			return nil, err
		}

		// get the ARN of the CMK used to encrypt the table
		keyArn, err := arn.Parse(*tableResult.Table.SSEDescription.KMSMasterKeyArn)
		if err != nil {
			return nil, err
		}

		accountID := e.Identity.Account
		roleARN := e.Identity.Role

		if keyArn.AccountID != accountID {
			return nil, fmt.Errorf("KMS key is in account %s, not account %s - iamzero does not yet support multi-account KMS setups", keyArn.AccountID, accountID)
		}

		svc := kms.NewFromConfig(cfg)

		_, err = svc.ListKeyPolicies(ctx, &kms.ListKeyPoliciesInput{
			KeyId: aws.String(keyArn.String()),
		})
		if err != nil {
			return nil, err
		}

		rec := KMSRecommendation{
			AccountID: accountID,
			RoleARN:   roleARN,
			KeyARN:    keyArn.String(),
			ID:        uuid.NewString(),
			Comment:   "Update the policy of the KMS key encrypting the DynamoDB table to allow access to the role",
		}

		return &rec, nil
	}
	return nil, nil
}

// Apply the recommendation by creating a grant for the KMS key
func (r *KMSRecommendation) Apply(log *zap.SugaredLogger) error {
	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	// Create service client value configured for credentials
	// from assumed role.
	svc := kms.NewFromConfig(cfg)

	grant := r.kmsGrantInput()
	// create a KMS grant allowing access to the key
	_, err = svc.CreateGrant(ctx, &grant)
	if err != nil {
		return err
	}

	return nil
}

func (r *KMSRecommendation) kmsGrantInput() kms.CreateGrantInput {
	return kms.CreateGrantInput{
		GranteePrincipal: aws.String(r.RoleARN),
		KeyId:            &r.KeyARN,
		Operations: []types.GrantOperation{
			types.GrantOperationDecrypt,
			types.GrantOperationEncrypt,
			types.GrantOperationReEncryptFrom,
			types.GrantOperationReEncryptTo,
			types.GrantOperationDescribeKey,
			types.GrantOperationGenerateDataKeyPair,
			types.GrantOperationGenerateDataKeyPairWithoutPlaintext,
		},
		Name: aws.String("iamzero"),
	}
}

func (r *KMSRecommendation) getDescription() []Description {
	grant := r.kmsGrantInput()

	desc := []Description{
		{
			AppliedTo: r.KeyARN,
			Type:      "KMS Grant",
			Policy:    grant,
		},
		{
			AppliedTo: r.RoleARN,
			Type:      "IAM Policy",
			Policy: events.AWSIAMPolicy{
				Version: "2012-10-17",
				Statement: []events.AWSIAMStatement{
					{
						Sid:    "iamzero",
						Effect: "Allow",
						Action: []string{
							"kms:Decrypt",
							"kms:Encrypt",
							"kms:ReEncryptFrom",
							"kms:ReEncryptTo",
							"kms:DescribeKey",
							"kms:GenerateDataKeyPair",
							"kms:GenerateDataKeyPairWithoutPlaintext",
						},
						Resource: []string{r.KeyARN},
					},
				},
			},
		},
	}

	return desc
}

func (r *KMSRecommendation) GetID() string {
	return r.ID
}

func (r *KMSRecommendation) Details() RecommendationDetails {
	desc := r.getDescription()
	details := RecommendationDetails{
		ID:          r.ID,
		Comment:     r.Comment,
		Description: desc,
	}
	return details
}

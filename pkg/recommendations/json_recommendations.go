package recommendations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type JSONPolicyParams struct {
	Policy  []Statement
	Comment string
	DocLink string
}

type Statement struct {
	Action   []string
	Resource []string
}

type JSONAdvice struct {
	ID        string
	AWSPolicy AWSIAMPolicy
	Comment   string
	RoleName  string
}

func GetJSONAdvice(r JSONPolicyParams) AdviceFactory {
	return func(e AWSEvent) (Advice, error) {

		var iamStatements []AWSIAMStatement

		// extract variables from the API call to insert in our recommended policies
		vars := e.Data.Parameters

		// include the region and account as variables available for templating
		// TODO: need to test this and consider edge cases where Parameters could contain a separate Region variable!
		vars["Region"] = e.Data.Region
		vars["Account"] = e.Identity.Account

		// generate AWS statements for each template statement we have
		for _, statement := range r.Policy {
			var resources []string

			for _, resourceTemplate := range statement.Resource {
				// template out each resource
				tmpl, err := template.New("policy").Parse(resourceTemplate)
				if err != nil {
					return nil, err
				}

				var resBytes bytes.Buffer
				err = tmpl.Execute(&resBytes, vars)
				if err != nil {
					return nil, err
				}
				resources = append(resources, resBytes.String())
			}

			iamStatement := AWSIAMStatement{

				Sid:      "iamzero-" + uuid.NewString(),
				Effect:   "Allow",
				Action:   statement.Action,
				Resource: resources,
			}
			iamStatements = append(iamStatements, iamStatement)
		}

		id := uuid.NewString()

		// build a recommended AWS policy
		policy := AWSIAMPolicy{
			Version:   "2012-10-17",
			Id:        &id,
			Statement: iamStatements,
		}

		roleName, err := GetRoleOrUserNameFromARN(e.Identity.Role)
		if err != nil {
			return nil, err
		}

		advice := JSONAdvice{
			AWSPolicy: policy,
			Comment:   r.Comment,
			ID:        id,
			RoleName:  roleName,
		}
		return &advice, nil
	}
}

// Apply the recommendation by provisioning and attaching an IAM policy to the role
func (a *JSONAdvice) Apply(log *zap.SugaredLogger) error {
	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	policyBytes, err := json.Marshal(a.AWSPolicy)
	if err != nil {
		return err
	}
	policyStr := string(policyBytes)
	name := fmt.Sprintf("iamzero-%s", a.ID)

	tagKey := "iamzero.dev/managed"
	tagVal := "true"

	svc := iam.NewFromConfig(cfg)

	params := iam.CreatePolicyInput{
		PolicyDocument: &policyStr,
		PolicyName:     &name,
		Tags: []types.Tag{
			{Key: &tagKey, Value: &tagVal},
		},
	}
	log.With("policy", params).Info("creating policy")

	createPolicyOutput, err := svc.CreatePolicy(ctx, &params)
	if err != nil {
		return err
	}

	arn := createPolicyOutput.Policy.Arn

	attachPolicyInput := iam.AttachRolePolicyInput{
		PolicyArn: arn,
		RoleName:  &a.RoleName,
	}

	_, err = svc.AttachRolePolicy(ctx, &attachPolicyInput)
	return err
}

func (a *JSONAdvice) GetID() string {
	return a.ID
}

func (a *JSONAdvice) getDescription() []Description {
	return []Description{
		{
			AppliedTo: a.RoleName,
			Type:      "IAM Policy",
			Policy:    a.AWSPolicy,
		},
	}
}

func (a *JSONAdvice) Details() RecommendationDetails {
	desc := a.getDescription()
	details := RecommendationDetails{
		ID:          a.ID,
		Comment:     a.Comment,
		Description: desc,
	}
	return details
}

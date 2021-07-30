package recommendations

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"strings"
	"text/template/parse"

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
	Resources []Resource
}

func GetJSONAdvice(r JSONPolicyParams) AdviceFactory {
	return func(e AWSEvent) (Advice, error) {

		var iamStatements []AWSIAMStatement
		resources := []Resource{}

		// extract variables from the API call to insert in our recommended policies
		vars := e.Data.Parameters

		// include the region and account as variables available for templating
		// TODO: need to test this and consider edge cases where Parameters could contain a separate Region variable!
		vars["Region"] = e.Data.Region
		vars["Account"] = e.Identity.Account

		// generate AWS statements for each template statement we have
		for _, statement := range r.Policy {
			// the actual ARN of the resource after executing the template
			// for example, `arn:aws:s3:::test-bucket/test-object`
			var renderedResources []string

			for _, resourceTemplate := range statement.Resource {
				// template out each resource
				tmpl, err := template.New("policy").Parse(resourceTemplate)
				if err != nil {
					return nil, err
				}

				friendlyResourceName, err := parseResourceFromTemplate(tmpl, vars)
				if err != nil {
					friendlyResourceName = "unknown"
				}
				resources = append(resources, Resource{
					ID:   uuid.NewString(),
					Name: friendlyResourceName,
				})

				var resBytes bytes.Buffer
				err = tmpl.Execute(&resBytes, vars)
				if err != nil {
					return nil, err
				}
				renderedResources = append(renderedResources, resBytes.String())
			}

			iamStatement := AWSIAMStatement{

				Sid:      "iamzero-" + uuid.NewString(),
				Effect:   "Allow",
				Action:   statement.Action,
				Resource: renderedResources,
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
			Resources: resources,
		}
		return &advice, nil
	}
}

// Apply the recommendation by provisioning and attaching an IAM policy to the role
// Note: we currently don't expose this functionality through the IAM Zero console and could consider
// removing it for now until we further explore use cases around IAM role deployment.
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
		Resources:   a.Resources,
		Description: desc,
	}
	return details
}

// parseResourceFromTemplate parses the resource out of a templated advisory.
// For example, if "arn:aws:s3:::{{ .Bucket }}/{{ .Key }}" is the template
// and the provided variables are Bucket=test-bucket and Key=test-key,
// the returned resource string should be test-bucket/test-key
//
// This method ignores the Account and Region fields to provide a human-friendly for the parsed resource
func parseResourceFromTemplate(t *template.Template, vars map[string]interface{}) (string, error) {
	resourceVals := []string{}

	// ignore the error here, we just try and match any variables used with the provided ones
	varsUsed, _ := requiredTemplateVars(t)
	for _, v := range varsUsed {
		if v == "Account" || v == "Region" {
			// don't include the Account or Region variables so that our
			// resource name is human-friendly
			continue
		}
		str, ok := vars[v].(string)
		if ok {
			resourceVals = append(resourceVals, str)
		}
	}
	if len(resourceVals) == 0 {
		return "", errors.New("resource could not be parsed")
	}
	resource := strings.Join(resourceVals, "/")
	return resource, nil
}

// Extract the template vars required from *simple* templates.
// Only works for top level, plain variables. Returns all problematic parse.Node as errors.
// Reference: https://stackoverflow.com/a/62224127
func requiredTemplateVars(t *template.Template) ([]string, []error) {
	var res []string
	var errors []error
	ln := t.Tree.Root
Node:
	for _, n := range ln.Nodes {
		if nn, ok := n.(*parse.ActionNode); ok {
			p := nn.Pipe
			if len(p.Decl) > 0 {
				errors = append(errors, fmt.Errorf("Node %v not supported", n))
				continue Node
			}
			for _, c := range p.Cmds {
				if len(c.Args) != 1 {
					errors = append(errors, fmt.Errorf("Node %v not supported", n))
					continue Node
				}
				if a, ok := c.Args[0].(*parse.FieldNode); ok {
					if len(a.Ident) != 1 {
						errors = append(errors, fmt.Errorf("Node %v not supported", n))
						continue Node
					}
					res = append(res, a.Ident[0])
				} else {
					errors = append(errors, fmt.Errorf("Node %v not supported", n))
					continue Node
				}

			}
		} else {
			if _, ok := n.(*parse.TextNode); !ok {
				errors = append(errors, fmt.Errorf("Node %v not supported", n))
				continue Node
			}
		}
	}
	return res, errors
}

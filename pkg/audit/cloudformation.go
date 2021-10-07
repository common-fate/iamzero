package audit

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/common-fate/iamzero/pkg/policies"
	"gopkg.in/yaml.v3"
)

func (a *Auditor) LoadCloudFormationStacks(ctx context.Context) error {
	if len(a.auditRoles) == 0 {
		return errors.New("no audit roles supplied")
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}
	svc := sts.NewFromConfig(cfg)

	for _, role := range a.auditRoles {
		a.log.With("role", role).Info("assuming role for audit")
		creds := stscreds.NewAssumeRoleProvider(svc, role)
		cfg.Credentials = aws.NewCredentialsCache(creds)

		// Create service client value configured for credentials
		// from assumed role.
		client := cloudformation.NewFromConfig(cfg)
		output, err := client.ListStacks(ctx, &cloudformation.ListStacksInput{})
		if err != nil {
			return err
		}

		cdkStacks := []types.StackSummary{}

		for _, stack := range output.StackSummaries {
			// extract the AWS account ID from the role
			stackARN, err := arn.Parse(*stack.StackId)
			if err != nil {
				return err
			}

			a.log.With("stack", stack.StackName).Debug("listing stack resources")
			resources, err := client.ListStackResources(ctx, &cloudformation.ListStackResourcesInput{
				StackName: stack.StackId,
			})
			if err != nil {
				return err
			}

			cdkResourcesInStack := []*policies.CDKResource{}
			isCDKStack := false

			for _, r := range resources.StackResourceSummaries {
				if *r.ResourceType == "AWS::CDK::Metadata" {
					// the CloudFormation stack has been defined using CDK
					cdkStacks = append(cdkStacks, stack)
					isCDKStack = true
				}
			}
			if isCDKStack {
				for _, r := range resources.StackResourceSummaries {
					a.log.With("resource", r).Debug("adding CDK resource")
					// the CDK metadata is not a real cloud resource like an IAM role or an S3 bucket,
					// so we don't worry about tracking it.
					if *r.ResourceType != "AWS::CDK::Metadata" {
						res := policies.CDKResource{
							Type:       *r.ResourceType,
							StackID:    *stack.StackId,
							LogicalID:  *r.LogicalResourceId,
							PhysicalID: *r.PhysicalResourceId,
							AccountID:  stackARN.AccountID,
						}
						cdkResourcesInStack = append(cdkResourcesInStack, &res)
					}
				}

				// we need to look up the raw template of the stack in order to find the CDK path metadata for
				// the resources defined in the CDK, in order to provide recommendations directly against
				// the CDK source code.
				tmpl, err := client.GetTemplate(ctx, &cloudformation.GetTemplateInput{
					StackName: stack.StackId,
				})
				if err != nil {
					return err
				}

				var obj CfnTemplate
				err = yaml.Unmarshal([]byte(*tmpl.TemplateBody), &obj)
				if err != nil {
					return err
				}

				for _, r := range cdkResourcesInStack {
					path := obj.Resources[r.LogicalID].Metadata.AwsCdkPath
					id := parseIDFromCDKPath(path)

					r.CDKPath = path
					r.CDKID = id

					a.cdkResources.Add(*r)
				}

				a.log.With("tmpl", obj).Debug("got template")
			}
		}
		a.log.With("cdkStacks", cdkStacks).Debug("found CDK stacks")
		// a.log.With("cdkResources", cdkResources).Debug("found CDK resources")
	}
	return nil
}

// parseIDFromCDKPath gets the CDK ID of a resource (as used in CDK source code)
// from a CDK path.
//
// An example CDK path is "CdkExampleStack/iamzero-overprivileged-role/Resource".
// The ID that we extract from this path is "iamzero-overprivileged-role"
//
// Returns an empty string if parsing fails.
func parseIDFromCDKPath(path string) string {
	split := strings.Split(path, "/")
	if len(split) < 1 {
		return ""
	}
	// the last entry in the path is "Resource", so we want to take the second last.
	return split[len(split)-2]
}

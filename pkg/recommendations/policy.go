package recommendations

import (
	"time"

	"github.com/common-fate/iamzero/pkg/policies"
	"github.com/common-fate/iamzero/pkg/tokens"
	"go.uber.org/zap"
)

const (
	PolicyStatusActive   = "active"
	PolicyStatusResolved = "resolved"
)

// Policy is a least-privilege policy generated by IAM Zero
type Policy struct {
	ID               string                `json:"id" storm:"id"`
	Identity         ProcessedAWSIdentity  `json:"identity"`
	LastUpdated      time.Time             `json:"lastUpdated"`
	Token            *tokens.Token         `json:"token"`
	EventCount       int                   `json:"eventCount"`
	Document         policies.AWSIAMPolicy `json:"document"`
	CDKFinding       *CDKFinding           `json:"cdkFinding"`
	TerraformFinding *TerraformFinding     `json:"terraformFinding"`
	// Status is either "active" or "resolved"
	Status string `json:"status"`
}

// ProcessedAWSIdentity is the same as AWS identity but contains optional
// definitions for infrastructure-as-code references to the identity
//
// This is used by IAM Zero to inform the user that we have matched a role
// that we've received actions for, with a reference to the said role in a
// CDK/Terraform stack that the user has deployed.
type ProcessedAWSIdentity struct {
	User        string                `json:"user"`
	Role        string                `json:"role"`
	Account     string                `json:"account"`
	CDKResource *policies.CDKResource `json:"cdkResource"`
}

// RecalculateDocument rebuilds the policy document based on the actions
// this initial implementation is naive and doesn't deduplicate or aggregate policies.
func (p *Policy) RecalculateDocument(actions []AWSAction) {
	statements := []policies.AWSIAMStatement{}

	for _, alert := range actions {
		if alert.Enabled && len(alert.Recommendations) > 0 {
			advisory := alert.GetSelectedAdvisory()
			for _, description := range advisory.Details().Description {
				// TODO: this should be redesigned to avoid casting from the interface.
				policy, ok := description.Policy.(policies.AWSIAMPolicy)
				if ok {
					statements = append(statements, policy.Statement...)
				}
			}
		}
	}

	p.LastUpdated = time.Now()
	p.EventCount = len(actions)
	p.Document.Statement = statements
}

func (p *Policy) RecalculateCDKFinding(actions []AWSAction, log *zap.SugaredLogger) {
	// only derive a CDK finding if we know that the role that we are
	// giving recommendations for has been defined using CDK
	if p.Identity.CDKResource == nil {
		return
	}
	f := CDKFinding{
		FindingID: p.ID,
		Role: CDKRole{
			Type:    p.Identity.CDKResource.Type,
			CDKPath: p.Identity.CDKResource.CDKPath,
		},
		Recommendations: []CDKRecommendation{},
	}

	for _, alert := range actions {
		if alert.Enabled && len(alert.Recommendations) > 0 {
			rec := CDKRecommendation{
				Type:       "IAMInlinePolicy",
				Statements: []CDKStatement{},
			}
			advisory := alert.GetSelectedAdvisory()
			for _, description := range advisory.Details().Description {
				// TODO: this should be redesigned to avoid casting from the interface.
				policy, ok := description.Policy.(policies.AWSIAMPolicy)
				log.With("ok", ok).Debug("found policy")
				if ok {
					for _, s := range policy.Statement {
						cdkStatement := CDKStatement{
							Actions: s.Action,
						}
						// TODO: we need to better structure resources so that
						// we have a reference to a CDK resource in an IAM statement
						for _, resource := range alert.Resources {

							var cdkResource CDKResource
							if resource.CDKResource != nil {
								cdkResource = CDKResource{
									Reference: "CDK",
									Type:      resource.CDKResource.Type,
									CDKPath:   &resource.CDKResource.CDKPath,
								}
							} else {
								cdkResource = CDKResource{
									Reference: "IAM",
									ARN:       &resource.ARN,
								}
							}
							cdkStatement.Resources = append(cdkStatement.Resources, cdkResource)
						}
						rec.Statements = append(rec.Statements, cdkStatement)
					}
				}
			}
			f.Recommendations = append(f.Recommendations, rec)
		}
	}

	p.CDKFinding = &f
}

func PolicyStatusIsValid(status string) bool {
	return status == PolicyStatusActive || status == PolicyStatusResolved
}

func (p *Policy) RecalculateTerraformFinding(actions []AWSAction, log *zap.SugaredLogger) {
	terraformFinding := TerraformFinding{
		FindingID: p.ID,
		Role: TerraformRole{
			Name: p.Identity.Role,
		},
		Recommendations: []TerraformRecommendation{
			// {
			// 	Type: "IAMInlinePolicy",
			// 	Statements: []TerraformStatement{
			// 		{
			// 			Resources: []TerraformResource{
			// 				{
			// 					Reference: bucketArn,
			// 					Type:      "AWS::S3::Bucket", ARN: &bucketArn,
			// 				},
			// 			},
			// 			Actions: actionsDemo,
			// 		},
			// 	},
			// },
		},
	}

	// I copied this and modified it from the CDK example, it is subject to the same TODO comments as CDK above
	for _, alert := range actions {
		if alert.Enabled && len(alert.Recommendations) > 0 {
			rec := TerraformRecommendation{
				Type:       "IAMInlinePolicy",
				Statements: []TerraformStatement{},
			}
			advisory := alert.GetSelectedAdvisory()
			for _, description := range advisory.Details().Description {
				policy, ok := description.Policy.(policies.AWSIAMPolicy)
				log.With("ok", ok).Debug("found policy")
				if ok {
					for _, s := range policy.Statement {
						terraformStatement := TerraformStatement{
							Actions: s.Action,
						}
						for _, resource := range alert.Resources {

							var terraformResource TerraformResource
							if resource.CDKResource == nil {

								terraformResource = TerraformResource{
									Reference: "IAM",
									ARN:       &resource.ARN,
								}
							}
							terraformStatement.Resources = append(terraformStatement.Resources, terraformResource)
						}
						rec.Statements = append(rec.Statements, terraformStatement)
					}
				}
			}
			terraformFinding.Recommendations = append(terraformFinding.Recommendations, rec)
		}
	}
	p.TerraformFinding = &terraformFinding

}

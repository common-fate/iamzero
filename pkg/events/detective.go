package events

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/common-fate/iamzero/pkg/audit"
	"github.com/common-fate/iamzero/pkg/policies"
	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/common-fate/iamzero/pkg/storage"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Detective looks through events to create and update Findings
type Detective struct {
	log            *zap.SugaredLogger
	actionStorage  storage.ActionStorage
	findingStorage storage.FindingStorage
	auditor        *audit.Auditor
}

type DetectiveOpts struct {
	Log            *zap.SugaredLogger
	ActionStorage  storage.ActionStorage
	FindingStorage storage.FindingStorage
	Auditor        *audit.Auditor
}

// NewDetective creates and initialises a new Detective
func NewDetective(opts DetectiveOpts) *Detective {
	return &Detective{
		log:            opts.Log,
		actionStorage:  opts.ActionStorage,
		findingStorage: opts.FindingStorage,
		auditor:        opts.Auditor,
	}
}

func (c *Detective) AnalyseEvent(e recommendations.AWSEvent) (*recommendations.AWSAction, error) {

	advisor := recommendations.NewAdvisor(c.auditor)

	// if the event was captured from an assumed role, replace it with the IAM role ARN
	// TODO: we should store both the session ARN and the role ARN in this case
	iamRole, err := recommendations.ExtractRoleARNFromSession(e.Identity.Role)
	if err != nil {
		return nil, err
	}
	if iamRole != nil {
		e.Identity.Role = *iamRole
	}

	// see whether we have an infrastructure-as-code definition for the role in question
	roleARN, err := arn.Parse(e.Identity.Role)
	if err != nil {
		return nil, err
	}
	c.log.With("roleARN", roleARN.Resource).Debug("decoded role ARN")

	physicalID, err := audit.GetPhysicalIDFromARNResource(roleARN.Resource)
	if err != nil {
		return nil, err
	}

	cdkResource := c.auditor.GetCDKResourceByPhysicalID(physicalID)
	c.log.With("cdkResource", cdkResource, "physicalID", physicalID).Debug("looked up CDK resource")

	// try and find an existing finding
	finding, err := c.findingStorage.FindByRole(storage.FindByRoleQuery{
		Role:   e.Identity.Role,
		Status: recommendations.PolicyStatusActive,
	})
	if err != nil {
		return nil, err
	}
	if finding == nil {
		identity := recommendations.ProcessedAWSIdentity{
			User:        e.Identity.User,
			Role:        e.Identity.Role,
			Account:     e.Identity.Account,
			CDKResource: cdkResource,
		}

		// create a new policy for the token and role if it doesn't exist
		finding = &recommendations.Finding{
			ID:         uuid.NewString(),
			Identity:   identity,
			UpdatedAt:  time.Now(),
			EventCount: 0,
			Status:     "active",
			Document: policies.AWSIAMPolicy{
				Version:   "2012-10-17",
				Statement: []policies.AWSIAMStatement{},
			},
		}
	}

	advice, err := advisor.Advise(e)
	if err != nil {
		return nil, err
	} else {
		c.log.With("advice", advice).Info("matched advisor recommendation")
	}

	action := recommendations.AWSAction{
		ID:                 uuid.NewString(),
		FindingID:          finding.ID,
		Event:              e,
		Status:             recommendations.AlertActive,
		Time:               time.Now(),
		HasRecommendations: false,
		// Resources:          []recommendations.CloudResourceInstance{},
		Recommendations:                []*recommendations.LeastPrivilegePolicy{},
		Enabled:                        true,
		SelectedLeastPrivilegePolicyID: "",
	}

	if len(advice) > 0 {
		action.HasRecommendations = true
		action.Recommendations = advice
		// action.Resources = advice[0].Details().Resources // TODO: we should aggregate resources across different advisories
		action.SelectedLeastPrivilegePolicyID = advice[0].GetID()
	}

	c.log.With("action", action).Info("adding action")
	err = c.actionStorage.Add(action)
	if err != nil {
		return nil, err
	}

	actions, err := c.actionStorage.ListForPolicy(finding.ID)
	if err != nil {
		return nil, err
	}
	finding.RecalculateDocument(actions)

	err = c.findingStorage.CreateOrUpdate(*finding)
	if err != nil {
		return nil, err
	}
	return &action, nil
}

package app

import (
	"errors"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/common-fate/iamzero/api/io"
	"github.com/common-fate/iamzero/internal/middleware"
	"github.com/common-fate/iamzero/pkg/audit"
	"github.com/common-fate/iamzero/pkg/policies"
	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/common-fate/iamzero/pkg/storage"
	"github.com/common-fate/iamzero/pkg/tokens"
	"github.com/go-chi/chi"
	"github.com/google/uuid"

	chiMiddleware "github.com/go-chi/chi/middleware"
)

func (c *Collector) GetCollectorRoutes() *chi.Mux {
	router := chi.NewRouter()

	router.Route("/api/v1", func(r chi.Router) {
		r.Use(chiMiddleware.RequestID)
		r.Use(chiMiddleware.RealIP)
		r.Use(middleware.Logger(c.log.Desugar()))
		r.Use(chiMiddleware.Recoverer)
		r.Use(chiMiddleware.Timeout(10 * time.Second))
		r.Use(middleware.Tracing)

		r.Group(func(r chi.Router) {
			// check the token for the event collector endpoint
			r.Use(middleware.CollectorTokenAuth(c.tokenStore, c.log))
			r.Route("/events", func(r chi.Router) {
				r.Post("/", c.HTTPCreateEventBatchHandler)
			})
		})
	})

	return router
}

type CreateEventBatchResponse struct {
	AlertIDs []string `json:"alertIDs"`
}

// HTTPCreateEventBatchHandler creates a batch of events
// Design assumption - all events in a given batch are dispatched for the same token and role
func (c *Collector) HTTPCreateEventBatchHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var rec []recommendations.AWSEvent

	token, ok := middleware.TokenFromContext(ctx)
	if !ok {
		io.RespondError(ctx, c.log, w, errors.New("could not load token"))
		return
	}

	if token == nil {
		http.Error(w, "invalid token", http.StatusBadRequest)
		return
	}

	if err := io.DecodeJSONBody(w, r, &rec); err != nil {
		io.RespondError(ctx, c.log, w, err)
		return
	}

	c.log.With("events", rec).Info("received events")

	advisor := recommendations.NewAdvisor(c.auditor)

	var res CreateEventBatchResponse
	for _, e := range rec {
		action, err := c.handleRecommendation(handleRecommendationArgs{
			Event:   e,
			Token:   token,
			Advisor: advisor,
		})

		if err != nil {
			io.RespondError(ctx, c.log, w, err)
			return
		}

		res.AlertIDs = append(res.AlertIDs, action.ID)
	}

	io.RespondJSON(ctx, c.log, w, res, http.StatusAccepted)
}

type handleRecommendationArgs struct {
	Event   recommendations.AWSEvent
	Token   *tokens.Token
	Advisor *recommendations.Advisor
}

// handleRecommendation handles a captured IAM Zero event and looks up advisories for it,
// saving results to the storage
func (c *Collector) handleRecommendation(args handleRecommendationArgs) (*recommendations.AWSAction, error) {
	e := args.Event
	advisor := args.Advisor
	token := args.Token

	// if the event was captured from an assumed role, replace it with the IAM role ARN
	// TODO: we should store both the session ARN and the role ARN in this case
	iamRole, err := recommendations.ExtractRoleARNFromSession(e.Identity.Role)
	if err != nil {
		return nil, err
	}
	if iamRole != nil {
		e.Identity.Role = *iamRole
	}

	// censor info if in demo mode
	if c.demo {
		e.Identity.User = "iamzero-test-user"
		e.Identity.Role = "arn:aws:iam::123456789012:role/iamzero-test-role"
		e.Identity.Account = "123456789012"
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

	// try and find an existing policy
	policy, err := c.policyStorage.FindByRole(storage.FindByRoleQuery{
		Role:   e.Identity.Role,
		Status: recommendations.PolicyStatusActive,
	})
	if err != nil {
		return nil, err
	}
	if policy == nil {
		identity := recommendations.ProcessedAWSIdentity{
			User:        e.Identity.User,
			Role:        e.Identity.Role,
			Account:     e.Identity.Account,
			CDKResource: cdkResource,
		}

		// create a new policy for the token and role if it doesn't exist
		policy = &recommendations.Policy{
			ID:          uuid.NewString(),
			Identity:    identity,
			LastUpdated: time.Now(),
			Token:       token,
			EventCount:  0,
			Status:      "active",
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
		PolicyID:           policy.ID,
		Event:              e,
		Status:             recommendations.AlertActive,
		Time:               time.Now(),
		HasRecommendations: false,
		Resources:          []recommendations.Resource{},
		Recommendations:    []*recommendations.JSONAdvice{},
		Enabled:            true,
		SelectedAdvisoryID: "",
	}

	if len(advice) > 0 {
		action.HasRecommendations = true
		action.Recommendations = advice
		action.Resources = advice[0].Details().Resources // TODO: we should aggregate resources across different advisories
		action.SelectedAdvisoryID = advice[0].GetID()
	}

	c.log.With("action", action).Info("adding action")
	err = c.actionStorage.Add(action)
	if err != nil {
		return nil, err
	}

	actions, err := c.actionStorage.ListForPolicy(policy.ID)
	if err != nil {
		return nil, err
	}
	policy.RecalculateDocument(actions)

	// if c.CDK {
	// 	policy.RecalculateCDKFinding(actions, c.log)
	// }

	policy.RecalculateTerraformFinding(actions, c.log)

	err = c.policyStorage.CreateOrUpdate(*policy)
	if err != nil {
		return nil, err
	}
	return &action, nil
}

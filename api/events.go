package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/common-fate/iamzero/api/io"
	"github.com/common-fate/iamzero/internal/middleware"
	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/common-fate/iamzero/pkg/storage"
	"github.com/common-fate/iamzero/pkg/tokens"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Handlers is the configuration for the handlers with required objects
type Handlers struct {
	Log           *zap.SugaredLogger
	TokenStore    tokens.TokenStorer
	ActionStorage *storage.ActionStorage
	PolicyStorage *storage.PolicyStorage
	Demo          bool
}

type CreateEventBatchResponse struct {
	AlertIDs []string `json:"alertIDs"`
}

// CreateEventBatch creates a batch of events
// Design assumption - all events in a given batch are dispatched for the same token and role
func (h *Handlers) CreateEventBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var rec []recommendations.AWSEvent

	token, ok := middleware.TokenFromContext(ctx)
	if !ok {
		io.RespondError(ctx, h.Log, w, errors.New("could not load token"))
		return
	}

	if err := io.DecodeJSONBody(w, r, &rec); err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	h.Log.With("events", rec).Info("received events")

	advisor := recommendations.NewAdvisor()

	var res CreateEventBatchResponse
	for _, e := range rec {
		// censor info if in demo mode
		if h.Demo {
			e.Identity.User = "iamzero-test-user"
			e.Identity.Role = "arn:aws:iam::123456789012:role/iamzero-test-role"
			e.Identity.Account = "123456789012"
		}

		// try and find an existing policy
		policy := h.PolicyStorage.FindByRoleAndToken(storage.FindPolicyQuery{
			Role:   e.Identity.Role,
			Token:  token.ID,
			Status: recommendations.PolicyStatusActive,
		})
		if policy == nil {
			// create a new policy for the token and role if it doesn't exist
			policy = &recommendations.Policy{
				ID:          uuid.NewString(),
				Identity:    e.Identity,
				LastUpdated: time.Now(),
				Token:       *token,
				EventCount:  0,
				Status:      "active",
				Document: recommendations.AWSIAMPolicy{
					Version:   "2012-10-17",
					Statement: []recommendations.AWSIAMStatement{},
				},
			}
		}

		advice, err := advisor.Advise(e)
		if err != nil {
			io.RespondError(ctx, h.Log, w, err)
			return
		} else {
			h.Log.With("advice", advice).Info("matched advisor recommendation")
		}

		action := recommendations.AWSAction{
			ID:                 uuid.NewString(),
			PolicyID:           policy.ID,
			Event:              e,
			Status:             recommendations.AlertActive,
			Time:               time.Now(),
			HasRecommendations: false,
			Enabled:            true,
			SelectedAdvisoryID: "",
		}

		res.AlertIDs = append(res.AlertIDs, action.ID)

		if len(advice) > 0 {
			action.HasRecommendations = true
			action.Recommendations = advice
			action.Resources = advice[0].Details().Resources // TODO: we should aggregate resources across different advisories
			action.SelectedAdvisoryID = advice[0].GetID()
		}

		h.Log.With("action", action).Info("adding action")
		h.ActionStorage.Add(action)

		actions := h.ActionStorage.ListForPolicy(policy.ID)
		policy.RecalculateDocument(actions)

		err = h.PolicyStorage.CreateOrUpdate(*policy)
		if err != nil {
			io.RespondError(ctx, h.Log, w, err)
			return
		}
	}

	io.RespondJSON(ctx, h.Log, w, res, http.StatusAccepted)
}

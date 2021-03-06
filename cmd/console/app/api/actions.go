package api

import (
	"net/http"
	"time"

	"github.com/common-fate/iamzero/api/io"
	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/go-chi/chi"
)

type ActionResponse struct {
	ID        string                                  `json:"id"`
	FindingID string                                  `json:"findingId"`
	Event     recommendations.AWSEvent                `json:"event"`
	Status    string                                  `json:"status"`
	Time      time.Time                               `json:"time"`
	Resources []recommendations.CloudResourceInstance `json:"resources"`

	Recommendations    []recommendations.RecommendationDetails `json:"recommendations"`
	HasRecommendations bool                                    `json:"hasRecommendations"`
}

// buildActionResponse loops through the advisories associated with an action
// to build a response
func buildActionResponse(action recommendations.AWSAction) ActionResponse {
	var detailsArr []recommendations.RecommendationDetails
	for _, rec := range action.Recommendations {
		details := rec.Details()
		detailsArr = append(detailsArr, details)
	}
	return ActionResponse{
		ID:        action.ID,
		FindingID: action.FindingID,
		Event:     action.Event,
		Status:    action.Status,
		// Resources:          action.Resources,
		Time:               action.Time,
		Recommendations:    detailsArr,
		HasRecommendations: action.HasRecommendations,
	}
}

func (h *Handlers) ListActions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actionsResponse := []ActionResponse{}

	actions, err := h.Storage.Action.List()
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	for _, action := range actions {
		res := buildActionResponse(action)
		actionsResponse = append(actionsResponse, res)
	}

	io.RespondJSON(ctx, h.Log, w, actionsResponse, http.StatusOK)
}

func (h *Handlers) GetAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actionID := chi.URLParam(r, "actionID")

	action, err := h.Storage.Action.Get(actionID)
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	if action == nil {
		http.Error(w, "action not found", http.StatusNotFound)
		return
	}

	res := buildActionResponse(*action)

	io.RespondJSON(ctx, h.Log, w, res, http.StatusOK)
}

type editActionBody struct {
	Enabled            *bool   `json:"enabled"`
	SelectedAdvisoryID *string `json:"selectedAdvisoryId"`
}

func (h *Handlers) EditAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actionID := chi.URLParam(r, "actionID")
	var b editActionBody

	if err := io.DecodeJSONBody(w, r, &b); err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	action, err := h.Storage.Action.Get(actionID)
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}
	if action == nil {
		io.RespondText(ctx, h.Log, w, "action not found", http.StatusNotFound)
		return
	}

	policy, err := h.Storage.Finding.Get(action.FindingID)
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}
	if policy == nil {
		io.RespondText(ctx, h.Log, w, "policy not found", http.StatusNotFound)
		return
	}

	if b.Enabled != nil {
		action.Enabled = *b.Enabled
	}

	if b.SelectedAdvisoryID != nil {
		if err := action.SelectAdvisory(*b.SelectedAdvisoryID); err != nil {
			io.RespondError(ctx, h.Log, w, err)
			return
		}
	}

	if err := h.Storage.Action.Update(*action); err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	// return the updated Policy corresponding to this alert
	actions, err := h.Storage.Action.ListForPolicy(policy.ID)
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	policy.RecalculateDocument(actions)
	if err := h.Storage.Finding.CreateOrUpdate(*policy); err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	io.RespondJSON(ctx, h.Log, w, policy, http.StatusOK)
}

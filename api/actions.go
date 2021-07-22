package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/common-fate/iamzero/api/io"
	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/go-chi/chi"
)

type ActionResponse struct {
	ID     string                   `json:"id"`
	Event  recommendations.AWSEvent `json:"event"`
	Status string                   `json:"status"`
	Time   time.Time                `json:"time"`

	Recommendations    []recommendations.RecommendationDetails `json:"recommendations"`
	HasRecommendations bool                                    `json:"hasRecommendations"`
}

func (h *Handlers) ListActions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	alertsResponse := []ActionResponse{}

	alerts := h.ActionStorage.List()

	for _, alert := range alerts {
		var detailsArr []recommendations.RecommendationDetails
		for _, rec := range alert.Recommendations {
			details := rec.Details()
			detailsArr = append(detailsArr, details)
		}
		alertRes := ActionResponse{
			ID:                 alert.ID,
			Event:              alert.Event,
			Status:             alert.Status,
			Time:               alert.Time,
			Recommendations:    detailsArr,
			HasRecommendations: alert.HasRecommendations,
		}
		alertsResponse = append(alertsResponse, alertRes)
	}

	io.RespondJSON(ctx, h.Log, w, alertsResponse, http.StatusOK)
}

type reviewActionBody struct {
	// "apply" or "ignore"
	Decision         string
	RecommendationID *string
}

// TODO: deprecated and to be removed.
func (h *Handlers) ReviewAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	alertID := chi.URLParam(r, "alertID")
	var b reviewActionBody

	if err := io.DecodeJSONBody(w, r, &b); err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	h.Log.With("body", b).Info("review alert")

	alert := h.ActionStorage.Get(alertID)
	if alert == nil {
		io.RespondText(ctx, h.Log, w, "alert not found", http.StatusNotFound)
		return
	}

	if b.Decision == "apply" {
		err := h.ActionStorage.SetStatus(alertID, recommendations.AlertApplying)
		if err != nil {
			io.RespondError(ctx, h.Log, w, errors.New("alert setstatus error"))
		}

		// TODO: make this more resilient
		go func() {
			ctx := context.Background()
			var recommendation recommendations.Advice

			for _, rec := range alert.Recommendations {
				if rec.GetID() == *b.RecommendationID {
					recommendation = rec
					break
				}
			}

			if recommendation == nil {
				io.RespondText(ctx, h.Log, w, "recommendation not found", http.StatusNotFound)
				return
			}

			err = recommendation.Apply(h.Log)
			if err != nil {
				io.RespondError(ctx, h.Log, w, errors.New("applier error"))
			}

			err = h.ActionStorage.SetStatus(alertID, recommendations.AlertFixed)
			if err != nil {
				io.RespondError(ctx, h.Log, w, errors.New("alert setstatus error"))
			}
		}()
		return

	} else if b.Decision == "ignore" {
		err := h.ActionStorage.SetStatus(alertID, recommendations.AlertIgnored)
		if err != nil {
			io.RespondError(ctx, h.Log, w, errors.New("alert setstatus error"))
		}
	}
}

type editActionBody struct {
	Enabled            *bool   `json:"enabled"`
	SelectedAdvisoryID *string `json:"selectedAdvisoryId"`
}

func (h *Handlers) EditAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	alertID := chi.URLParam(r, "alertID")
	var b editActionBody

	if err := io.DecodeJSONBody(w, r, &b); err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	action := h.ActionStorage.Get(alertID)
	if action == nil {
		io.RespondText(ctx, h.Log, w, "action not found", http.StatusNotFound)
		return
	}

	policy := h.PolicyStorage.Get(action.PolicyID)
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

	if err := h.ActionStorage.Update(*action); err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	// return the updated Policy corresponding to this alert
	actions := h.ActionStorage.ListForPolicy(policy.ID)

	policy.RecalculateDocument(actions)
	if err := h.PolicyStorage.CreateOrUpdate(*policy); err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	io.RespondJSON(ctx, h.Log, w, policy, http.StatusOK)
}

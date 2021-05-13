package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/common-fate/iamzero/api/io"
	"github.com/common-fate/iamzero/pkg/events"
	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/go-chi/chi"
)

type AlertResponse struct {
	ID     string          `json:"id"`
	Event  events.AWSEvent `json:"event"`
	Status string          `json:"status"`
	Time   time.Time       `json:"time"`

	Recommendations    []recommendations.RecommendationDetails `json:"recommendations"`
	HasRecommendations bool                                    `json:"hasRecommendations"`
}

func (h *Handlers) ListAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	alertsResponse := []AlertResponse{}

	alerts := h.Storage.List()

	for _, alert := range alerts {
		var detailsArr []recommendations.RecommendationDetails
		for _, rec := range alert.Recommendations {
			details := rec.Details()
			detailsArr = append(detailsArr, details)
		}
		alertRes := AlertResponse{
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

type reviewAlertBody struct {
	// "apply" or "ignore"
	Decision         string
	RecommendationID *string
}

func (h *Handlers) ReviewAlert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	alertID := chi.URLParam(r, "alertID")
	var b reviewAlertBody

	if err := io.DecodeJSONBody(w, r, &b); err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	h.Log.With("body", b).Info("review alert")

	alert := h.Storage.Get(alertID)
	if alert == nil {
		io.RespondText(ctx, h.Log, w, "alert not found", http.StatusNotFound)
		return
	}

	if b.Decision == "apply" {
		err := h.Storage.SetStatus(alertID, events.AlertApplying)
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

			err = h.Storage.SetStatus(alertID, events.AlertFixed)
			if err != nil {
				io.RespondError(ctx, h.Log, w, errors.New("alert setstatus error"))
			}
		}()
		return

	} else if b.Decision == "ignore" {
		err := h.Storage.SetStatus(alertID, events.AlertIgnored)
		if err != nil {
			io.RespondError(ctx, h.Log, w, errors.New("alert setstatus error"))
		}
	}

}

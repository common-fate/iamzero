package server

import (
	"net/http"

	"github.com/common-fate/iamzero/api/io"
	"github.com/common-fate/iamzero/pkg/events"
)

// CreateEventBatch creates a batch of events
func (h *Server) CreateEventBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var rec []events.AWSEvent

	if err := io.DecodeJSONBody(w, r, &rec); err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	h.Log.With("events", rec).Info("received events")

	// advisor := recommendations.NewAdvisor()

	// var res CreateEventBatchResponse
	// for _, e := range rec {
	// 	// censor info if in demo mode
	// 	if h.Demo {
	// 		e.Identity.User = "iamzero-test-user"
	// 		e.Identity.Account = "123456789"
	// 	}

	// 	advice, err := advisor.Advise(e)
	// 	if err != nil {
	// 		io.RespondError(ctx, h.Log, w, err)
	// 		return
	// 	} else {
	// 		h.Log.With("advice", advice).Info("matched advisor recommendation")
	// 	}

	// 	alert := recommendations.AWSAlert{
	// 		ID:                 uuid.NewString(),
	// 		Event:              e,
	// 		Status:             events.AlertActive,
	// 		Time:               time.Now(),
	// 		HasRecommendations: false,
	// 	}

	// 	res.AlertIDs = append(res.AlertIDs, alert.ID)

	// 	if len(advice) > 0 {
	// 		alert.HasRecommendations = true
	// 		alert.Recommendations = advice
	// 	}

	// 	h.Log.With("alert", alert).Info("adding alert")
	// 	h.Storage.Add(alert)
	// }

	var res struct{}

	io.RespondJSON(ctx, h.Log, w, res, http.StatusAccepted)
}

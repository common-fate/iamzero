package app

import (
	"errors"
	"net/http"
	"time"

	"github.com/common-fate/iamzero/api/io"
	"github.com/common-fate/iamzero/internal/middleware"
	"github.com/common-fate/iamzero/pkg/events"
	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/go-chi/chi"

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

	detective := events.NewDetective(events.DetectiveOpts{
		Log:     c.log,
		Storage: c.storage,
		Auditor: c.auditor,
	})

	var res CreateEventBatchResponse
	for _, e := range rec {
		// censor info if in demo mode
		if c.demo {
			e.Identity.User = "iamzero-test-user"
			e.Identity.Role = "arn:aws:iam::123456789012:role/iamzero-test-role"
			e.Identity.Account = "123456789012"
		}

		action, err := detective.AnalyseEvent(e)

		if err != nil {
			io.RespondError(ctx, c.log, w, err)
			return
		}

		res.AlertIDs = append(res.AlertIDs, action.ID)
	}

	io.RespondJSON(ctx, c.log, w, res, http.StatusAccepted)
}

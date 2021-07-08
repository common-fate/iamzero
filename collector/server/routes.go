package server

import (
	"net/http"
	"os"
	"time"

	"github.com/common-fate/iamzero/api"
	"github.com/common-fate/iamzero/internal/middleware"
	"github.com/go-chi/chi"
	chiMiddleware "github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
)

type Server struct {
	Shutdown chan os.Signal
	Log      *zap.SugaredLogger
	Token    string
	Demo     bool
}

// API constructs an http.Handler with all application routes defined.
func (s *Server) API() http.Handler {
	app := chi.NewRouter()

	// Register health check endpoint. This route is not authenticated.
	check := api.Check{Log: s.Log}
	app.Get("/api/v1/health", check.Health)

	app.Route("/api/v1", func(r chi.Router) {
		r.Use(chiMiddleware.RequestID)
		r.Use(chiMiddleware.RealIP)
		r.Use(middleware.Logger(s.Log.Desugar()))
		r.Use(chiMiddleware.Recoverer)
		r.Use(chiMiddleware.Timeout(60 * time.Second))
		r.Use(middleware.SimpleTokenAuth(s.Token))

		r.Post("/events/", s.CreateEventBatch)
	})

	return app
}

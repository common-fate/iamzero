package server

import (
	"net/http"
	"os"
	"time"

	"github.com/common-fate/iamzero/api"
	"github.com/common-fate/iamzero/internal/middleware"
	"github.com/common-fate/iamzero/pkg/storage"
	"github.com/common-fate/iamzero/web"
	"github.com/go-chi/chi"
	chiMiddleware "github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
)

// APIConfig is the configuration struct to build the API handlers
type APIConfig struct {
	Shutdown         chan os.Signal
	Log              *zap.SugaredLogger
	Token            string
	Demo             bool
	ProxyAuthEnabled bool
}

// API constructs an http.Handler with all application routes defined.
func API(cfg *APIConfig) http.Handler {

	// Construct the App which holds all routes as well as common Middleware.
	app := NewApp(cfg.Shutdown, cfg.Log)

	// Register health check endpoint. This route is not authenticated.
	check := api.Check{Log: cfg.Log}
	app.Get("/api/v1/health", check.Health)

	handlers := api.Handlers{
		Log:     cfg.Log,
		Demo:    cfg.Demo,
		Storage: storage.NewAlertStorage(),
	}

	// Main application routes
	app.Route("/api/v1", func(r chi.Router) {
		r.Use(chiMiddleware.RequestID)
		r.Use(chiMiddleware.RealIP)
		r.Use(middleware.Logger(cfg.Log.Desugar()))
		r.Use(chiMiddleware.Recoverer)
		r.Use(chiMiddleware.Timeout(60 * time.Second))

		r.Group(func(r chi.Router) {
			// check the token for the event collector endpoint, even if reverse-proxy auth is enabled
			r.Use(middleware.SimpleTokenAuth(cfg.Token))
			r.Route("/events", func(r chi.Router) {
				r.Post("/", handlers.CreateEventBatch)
			})
		})

		r.Group(func(r chi.Router) {
			// these routes are protected via reverse-proxy auth

			r.Route("/alerts", func(r chi.Router) {
				r.Get("/", handlers.ListAlerts)

				r.Route("/{alertID}", func(r chi.Router) {
					r.Post("/review", handlers.ReviewAlert)
				})
			})
		})
	})

	app.Route("/", func(r chi.Router) {
		staticHandler := web.AssetHandler("/", "build")

		r.Get("/", staticHandler.ServeHTTP)
		r.Get("/*", staticHandler.ServeHTTP)
	})

	return app
}

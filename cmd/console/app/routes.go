package app

import (
	"time"

	"github.com/common-fate/iamzero/cmd/console/app/api"
	"github.com/common-fate/iamzero/internal/middleware"
	"github.com/common-fate/iamzero/web"
	"github.com/go-chi/chi"

	chiMiddleware "github.com/go-chi/chi/middleware"
)

func (c *Console) GetConsoleRoutes() *chi.Mux {
	router := chi.NewRouter()
	handlers := api.Handlers{
		Log:            c.log,
		TokenStore:     c.tokenStore,
		ActionStorage:  c.actionStorage,
		FindingStorage: c.policyStorage,
		Auditor:        c.auditor,
	}

	router.Route("/api/v1", func(r chi.Router) {
		r.Use(chiMiddleware.RequestID)
		r.Use(chiMiddleware.RealIP)
		r.Use(middleware.Logger(c.log.Desugar()))
		r.Use(chiMiddleware.Recoverer)
		r.Use(chiMiddleware.Timeout(10 * time.Second))
		r.Use(middleware.Tracing)

		r.Group(func(r chi.Router) {
			r.Route("/tokens", func(r chi.Router) {
				r.Get("/", handlers.ListTokens)
				r.Post("/", handlers.CreateToken)
				r.Delete("/{tokenID}", handlers.DeleteToken)
			})

			r.Route("/actions", func(r chi.Router) {
				r.Get("/", handlers.ListActions)

				r.Route("/{actionID}", func(r chi.Router) {
					r.Get("/", handlers.GetAction)
					r.Put("/edit", handlers.EditAction)
				})
			})

			r.Route("/policies", func(r chi.Router) {
				r.Get("/", handlers.ListPolicies)
				r.Get("/find", handlers.FindFinding)
				r.Get("/{policyID}", handlers.GetFinding)
				r.Get("/{policyID}/actions", handlers.ListActionsForFinding)
				r.Put("/{policyID}/status", handlers.SetFindingStatus)
			})
		})
	})

	router.Route("/", func(r chi.Router) {
		staticHandler := web.AssetHandler("/", "build")

		r.Get("/", staticHandler.ServeHTTP)
		r.Get("/*", staticHandler.ServeHTTP)
	})

	return router
}

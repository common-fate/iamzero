package server

import (
	"net/http"
	"os"
	"syscall"

	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

// App is the entrypoint into our application and what configures our TODO: context
// object for each of our http handlers.
type App struct {
	*chi.Mux
	shutdown chan os.Signal
}

// NewApp creates an App value that handle a set of routes for the application.
func NewApp(shutdown chan os.Signal, log *zap.SugaredLogger) *App {
	app := App{
		Mux:      chi.NewRouter(),
		shutdown: shutdown,
	}

	return &app
}

// SignalShutdown is used to gracefully shutdown the app when an integrity
// issue is identified.
func (a *App) SignalShutdown() {
	a.shutdown <- syscall.SIGTERM
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// tracing can be wrapped here
	// a.och.ServeHTTP(w, r)
	a.Mux.ServeHTTP(w, r)
}

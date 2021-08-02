package service

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"

	chiMiddleware "github.com/go-chi/chi/middleware"

	"github.com/common-fate/iamzero/pkg/healthcheck"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	healthCheckHTTPPort = "health-check-http-port"
	adminHTTPPort       = "admin-http-port"
	adminHTTPHostPort   = "admin.http.host-port"
)

// AdminServer runs an HTTP server with admin endpoints, such as healthcheck at /, /metrics, etc.
type AdminServer struct {
	logger        *zap.Logger
	adminHostPort string

	hc *healthcheck.HealthCheck

	mux    *chi.Mux
	server *http.Server
}

// NewAdminServer creates a new admin server.
func NewAdminServer(hostPort string) *AdminServer {
	return &AdminServer{
		adminHostPort: hostPort,
		logger:        zap.NewNop(),
		hc:            healthcheck.New(),
		mux:           chi.NewRouter(),
	}
}

// HC returns the reference to HeathCheck.
func (s *AdminServer) HC() *healthcheck.HealthCheck {
	return s.hc
}

// AddFlags registers CLI flags.
func (s *AdminServer) AddFlags(flagSet *flag.FlagSet) {
	flagSet.String(adminHTTPHostPort, s.adminHostPort, fmt.Sprintf("The host:port (e.g. 127.0.0.1%s or %s) for the admin server, including health check, /metrics, etc.", s.adminHostPort, s.adminHostPort))
}

// Handle adds a new handler to the admin server.
func (s *AdminServer) Handle(path string, handler http.Handler) {
	s.mux.Handle(path, handler)
}

// Serve starts HTTP server.
func (s *AdminServer) Serve() error {
	l, err := net.Listen("tcp", s.adminHostPort)
	if err != nil {
		s.logger.Error("Admin server failed to listen", zap.Error(err))
		return err
	}
	s.serveWithListener(l)

	s.logger.Info(
		"Admin server started",
		zap.String("http.host-port", l.Addr().String()),
		zap.Stringer("health-status", s.hc.Get()))
	return nil
}

func (s *AdminServer) serveWithListener(l net.Listener) {
	s.logger.Info("Mounting health check on admin server", zap.String("route", "/"))
	s.mux.Use(chiMiddleware.Recoverer)
	s.mux.Handle("/", s.hc.Handler())
	s.registerPprofHandlers()

	errorLog, _ := zap.NewStdLogAt(s.logger, zapcore.ErrorLevel)

	s.server = &http.Server{
		Handler:  s.mux,
		ErrorLog: errorLog,
	}

	s.logger.Info("Starting admin HTTP server", zap.String("http-addr", s.adminHostPort))
	go func() {
		switch err := s.server.Serve(l); err {
		case nil, http.ErrServerClosed:
			// normal exit, nothing to do
		default:
			s.logger.Error("failed to serve", zap.Error(err))
			s.hc.Set(healthcheck.Broken)
		}
	}()
}

func (s *AdminServer) registerPprofHandlers() {
	s.mux.HandleFunc("/debug/pprof/", pprof.Index)
	s.mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	s.mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	s.mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	s.mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	s.mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	s.mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	s.mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	s.mux.Handle("/debug/pprof/block", pprof.Handler("block"))
}

// Close stops the HTTP server
func (s *AdminServer) Close() error {
	return s.server.Shutdown(context.Background())
}

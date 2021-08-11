package service

import (
	"flag"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/common-fate/iamzero/pkg/healthcheck"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Service is a base IAM Zero backend service with shared functionality
// such as logging and tracing
type Service struct {
	Logger *zap.SugaredLogger
	Tracer *trace.Tracer

	// AdminPort is the HTTP port number for admin server.
	AdminPort int

	// Admin is the admin server that hosts the health check and metrics endpoints.
	Admin *AdminServer

	signalsChannel  chan os.Signal
	hcStatusChannel chan healthcheck.Status

	logLevel string
	// the port to expose the healthcheck and metrics on. Default is 10866
	adminPort int
}

func NewService() *Service {
	signalsChannel := make(chan os.Signal, 1)
	hcStatusChannel := make(chan healthcheck.Status)
	signal.Notify(signalsChannel, os.Interrupt, syscall.SIGTERM)

	return &Service{
		signalsChannel:  signalsChannel,
		hcStatusChannel: hcStatusChannel,
	}
}

func (s *Service) AddFlags(fs *flag.FlagSet) {
	fs.IntVar(&s.adminPort, "admin-port", 10866, "the port to expose healthcheck and metrics on")
	fs.StringVar(&s.logLevel, "log-level", "info", "the log level (must match go.uber.org/zap log levels)")
}

func (s *Service) Start() error {
	s.Admin = NewAdminServer(portToHostPort(s.adminPort))

	cfg := zap.NewProductionConfig()
	err := cfg.Level.UnmarshalText([]byte(s.logLevel))
	if err != nil {
		return err
	}
	logProd, err := cfg.Build()

	if err != nil {
		return err
	}
	s.Logger = logProd.Sugar()

	if err := s.Admin.Serve(); err != nil {
		return errors.Wrap(err, "starting admin server")
	}

	return nil
}

func (s *Service) RunAndThen(shutdown func()) {
	s.Admin.HC().Set(healthcheck.Ready)
statusLoop:
	for {
		select {
		case status := <-s.hcStatusChannel:
			s.Admin.HC().Set(status)
		case <-s.signalsChannel:
			break statusLoop
		}
	}

	s.Logger.Info("shutting down")

	if shutdown != nil {
		shutdown()
	}

	s.Logger.Info("shutdown complete")
}

// portToHostPort converts the port into a host:port address string
func portToHostPort(port int) string {
	return ":" + strconv.Itoa(port)
}

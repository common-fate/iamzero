package service

import (
	"os"
	"strconv"

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
}

func NewService(adminPort int) *Service {
	signalsChannel := make(chan os.Signal, 1)
	hcStatusChannel := make(chan healthcheck.Status)
	return &Service{
		Admin:           NewAdminServer(portToHostPort(adminPort)),
		signalsChannel:  signalsChannel,
		hcStatusChannel: hcStatusChannel,
	}
}

func (s *Service) Start() error {
	logProd, err := zap.NewProduction()
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

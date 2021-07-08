package main

import (
	"context"
	"fmt"
	syslog "log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ardanlabs/conf"
	"github.com/common-fate/iamzero/collector/server"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var build = "dev"

func run() error {

	var cfg struct {
		Host            string        `conf:"default:0.0.0.0:9090"`
		ReadTimeout     time.Duration `conf:"default:5s"`
		WriteTimeout    time.Duration `conf:"default:5s"`
		ShutdownTimeout time.Duration `conf:"default:5s"`
		Token           string        `conf:""`
		Demo            bool          `conf:"default:false"`

		conf.Version
	}

	cfg.Version = conf.Version{
		SVN:  build,
		Desc: "collector",
	}

	logProd, err := zap.NewProduction()
	if err != nil {
		syslog.Fatalf("can't initialize zap logger: %v", err)
	}

	log := logProd.Sugar()
	defer func() {
		err = log.Sync()
	}()

	if err := conf.Parse(os.Args[1:], "IAMZERO", &cfg); err != nil {
		if err == conf.ErrHelpWanted {
			usage, err := conf.Usage("IAMZERO", &cfg)
			if err != nil {
				return errors.Wrap(err, "generating config usage")
			}
			fmt.Println(usage)
			return nil
		}
		if err == conf.ErrVersionWanted {
			version, err := conf.VersionString("APP", &cfg)
			if err != nil {
				return err
			}
			fmt.Println(version)
			return nil
		}
		return errors.Wrap(err, "parsing config")
	}

	// Simple authentication via IAMZERO_TOKEN env variable
	if cfg.Token == "" {
		syslog.Fatal("IAMZERO_TOKEN variable must be provided")
	}

	log = log.With(zap.String("ver", cfg.Version.SVN))

	log.With("token", cfg.Token).Info("token")

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a buffered channel because the signal package requires it.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	server := server.Server{
		Shutdown: shutdown,
		Log:      log,
		Demo:     cfg.Demo,
		Token:    cfg.Token,
	}

	api := http.Server{
		Addr:         cfg.Host,
		Handler:      server.API(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	log.With("host", cfg.Host).Info("Starting server")

	if cfg.Demo {
		log.Info("Running in DEMO mode. AWS details will be censored")
	}

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		log.Infof("main : API listening on %s", api.Addr)
		serverErrors <- api.ListenAndServe()
	}()

	// =========================================================================
	// Shutdown

	// Blocking main and waiting for shutdown.
	select {
	case err := <-serverErrors:
		return errors.Wrap(err, "server error")

	case sig := <-shutdown:
		log.Infof("main : %v : Start shutdown", sig)

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		// Asking listener to shutdown and load shed.
		err := api.Shutdown(ctx)
		if err != nil {
			log.Infof("main : Graceful shutdown did not complete in %v : %v", cfg.ShutdownTimeout, err)
			return api.Close()
		}
	}
	return err
}

func main() {
	if err := run(); err != nil {
		syslog.Println("error :", err)
		os.Exit(1)
	}
}

package commands

import (
	"context"
	"flag"
	"io"
	syslog "log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/common-fate/iamzero/cmd/server"
	"github.com/common-fate/iamzero/pkg/tokens"
	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// overriden by build flags
var version string

// ServerCommand configuration object
type ServerCommand struct {
	RootConfig                    *RootConfig
	Out                           io.Writer
	Host                          string
	Token                         string
	ReadTimeout                   time.Duration
	WriteTimeout                  time.Duration
	ShutdownTimeout               time.Duration
	Demo                          bool
	TokenStorageBackend           string
	TokenStorageDynamoDBTableName string
	ProxyAuthEnabled              bool
}

// NewServerCommand creates a new ffcli.Command
func NewServerCommand(rootConfig *RootConfig, out io.Writer) *ffcli.Command {
	cfg := ServerCommand{
		RootConfig: rootConfig,
		Out:        out,
		Demo:       false, // Demo flag is overridden in the `local` command, but is not exposed for the `server` command
	}

	fs := flag.NewFlagSet("iamzero server", flag.ExitOnError)
	fs.StringVar(&cfg.Host, "host", "0.0.0.0:9090", "the server hostname to listen on (can be set via IAMZERO_HOST env var)")
	fs.DurationVar(&cfg.ReadTimeout, "read-timeout", 5*time.Second, "server read timeout duration (can be set via IAMZERO_READ_TIMEOUT env var)")
	fs.DurationVar(&cfg.WriteTimeout, "write-timeout", 5*time.Second, "server write timeout duration (can be set via IAMZERO_WRITE_TIMEOUT env var)")
	fs.DurationVar(&cfg.ShutdownTimeout, "shutdown-timeout", 5*time.Second, "server shutdown timeout duration (can be set via IAMZERO_SHUTDOWN_TIMEOUT env var)")
	fs.StringVar(&cfg.Token, "token", "", "authentication token (can be set via IAMZERO_TOKEN env var)")
	fs.StringVar(&cfg.TokenStorageBackend, "token-storage-backend", "dynamodb", "token storage backend (must be 'dynamodb' or 'inmemory')")
	fs.StringVar(&cfg.TokenStorageDynamoDBTableName, "token-storage-dynamodb-table-name", "dynamodb", "the token storage table name (only for DynamoDB token storage backend)")
	fs.BoolVar(&cfg.ProxyAuthEnabled, "proxy-auth-enabled", false, "use a reverse proxy to handle user authentication")
	rootConfig.RegisterFlags(fs)

	return &ffcli.Command{
		Name:       "server",
		ShortUsage: "iamzero server [flags] [<prefix>]",
		ShortHelp:  "Start an iamzero server (for server usage)",
		FlagSet:    fs,
		// allow setting environment variables to configure server settings
		Options: []ff.Option{ff.WithEnvVarPrefix("IAMZERO")},
		Exec:    cfg.Exec,
	}
}

// Exec function for this command.
func (c *ServerCommand) Exec(ctx context.Context, _ []string) error {
	logProd, err := zap.NewProduction()
	if err != nil {
		syslog.Fatalf("can't initialize zap logger: %v", err)
	}

	// Simple authentication via IAMZERO_TOKEN env variable
	if c.Token == "" {
		syslog.Fatal("IAMZERO_TOKEN variable must be provided")
	}

	log := logProd.Sugar().With("ver", version)

	defer func() {
		err = log.Sync()
	}()

	// Configure token storage
	if c.TokenStorageBackend != "dynamodb" {
		syslog.Fatalf("token storage backend %s is not supported", c.TokenStorageBackend)
	}
	tokenStore, err := tokens.NewDynamoDBTokenStorer(ctx, c.TokenStorageDynamoDBTableName)
	if err != nil {
		return err
	}

	// Start the application

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a buffered channel because the signal package requires it.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	apiConfig := server.APIConfig{
		Shutdown:         shutdown,
		Log:              log,
		Demo:             c.Demo,
		Token:            c.Token,
		TokenStore:       tokenStore,
		ProxyAuthEnabled: c.ProxyAuthEnabled,
	}

	api := http.Server{
		Addr:         c.Host,
		Handler:      server.API(&apiConfig),
		ReadTimeout:  c.ReadTimeout,
		WriteTimeout: c.WriteTimeout,
	}

	log.With("host", c.Host).Info("Starting server")

	if apiConfig.Demo {
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
		ctx, cancel := context.WithTimeout(context.Background(), c.ShutdownTimeout)
		defer cancel()

		// Asking listener to shutdown and load shed.
		err := api.Shutdown(ctx)
		if err != nil {
			log.Infof("main : Graceful shutdown did not complete in %v : %v", c.ShutdownTimeout, err)
			return api.Close()
		}
	}
	return err
}

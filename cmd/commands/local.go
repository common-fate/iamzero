package commands

import (
	"context"
	"flag"
	"fmt"
	"io"
	syslog "log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/common-fate/iamzero/cmd/server"
	"github.com/common-fate/iamzero/pkg/tokens"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"gopkg.in/ini.v1"
)

// LocalCommand configuration object
type LocalCommand struct {
	rootConfig *RootConfig
	out        io.Writer
	port       int
	demo       bool
}

// LocalCommand creates a new ffcli.Command
func NewLocalCommand(rootConfig *RootConfig, out io.Writer) *ffcli.Command {
	cfg := LocalCommand{
		rootConfig: rootConfig,
		out:        out,
	}

	fs := flag.NewFlagSet("iamzero local", flag.ExitOnError)
	fs.IntVar(&cfg.port, "p", 9090, "the local port to run the iamzero server on")
	fs.BoolVar(&cfg.demo, "demo", false, "run in demo mode (censors AWS account information)")
	rootConfig.RegisterFlags(fs)

	return &ffcli.Command{
		Name:       "local",
		ShortUsage: "iamzero local [flags] [<prefix>]",
		ShortHelp:  "Run a local iamzero server",
		FlagSet:    fs,
		Exec:       cfg.Exec,
	}
}

func (c *LocalCommand) log(a ...interface{}) {
	fmt.Fprintln(c.out, a...)
}

// Exec function for this command.
func (c *LocalCommand) Exec(ctx context.Context, _ []string) error {

	logProd, err := zap.NewProduction()
	if err != nil {
		return errors.Wrap(err, "can't initialize zap logger")
	}

	log := logProd.Sugar().With("ver", version)

	defer func() {
		err = log.Sync()
		if err != nil {
			syslog.Fatalf("error closing log: %v", err)
		}
	}()

	tracer := trace.NewNoopTracerProvider().Tracer("")

	// iamzero writes config to ~/.iamzero.ini, to allow developers
	// to set consistent settings between different projects they work on

	// check whether an iamzero config file exists already
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	file := path.Join(home, ".iamzero.ini")

	url := "http://localhost:" + strconv.Itoa(c.port)

	// we use the inmemory token storage backend to allow users to test
	// IAM Zero without depending on external dependencies like caches or databases
	tokenStore := tokens.NewInMemoryTokenStorer(ctx, log, tracer)

	// put the token into our in-memory token storage so that the user can send events to IAM Zero
	// Note: in future the local version of IAM Zero could simply not use token storage at all,
	// our collector endpoint could just be unauthenticated.
	token, err := tokenStore.Create(ctx, "Local token")
	if err != nil {
		return err
	}

	if _, err := os.Stat(file); err == nil {
		c.log("Loading your iamzero config file (" + file + ")")
		// config file exists
		cfgFile, err := ini.Load(file)
		if err != nil {
			return err
		}
		cfgFile.Section("iamzero").Key("token").SetValue(token.ID)
		savedUrl := cfgFile.Section("iamzero").Key("url")

		if savedUrl.String() != url {
			c.log("The URL in your config file (" + savedUrl.String() + ") was different to the URL your local iamzero server will run on (" + url + "). Updating your config file URL to be " + url + "...")
			savedUrl.SetValue(url)
		}
		err = cfgFile.SaveTo(file)
		if err != nil {
			return err
		}

	} else if os.IsNotExist(err) {
		// config file does not exist
		c.log(file + " does not exist - initialising new config")
		if err != nil {
			return err
		}

		cfgFile := ini.Empty()
		cfgFile.Section("iamzero").Key("token").SetValue(token.ID)
		cfgFile.Section("iamzero").Key("url").SetValue(url)
		err = cfgFile.SaveTo(file)
		if err != nil {
			return err
		}

		c.log("A new token has been generated for your iamzero server. You can view the token and server URL settings at " + file)
		c.log("By default, any iamzero client libraries you run in this computer will use this configuration file, unless you override their settings through environment variables or by passing settings when initialising the library in your code.")

	} else {
		// unknown error
		return err
	}

	c.log("Running local version of iamzero - web console can be accessed at " + url)

	err = openBrowser(url)
	if err != nil {
		c.log("error opening browser: ", err.Error())
	}

	// Start the application

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a buffered channel because the signal package requires it.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	apiConfig := server.APIConfig{
		Shutdown:   shutdown,
		Log:        log,
		Tracer:     tracer,
		Demo:       c.demo,
		TokenStore: tokenStore,
	}

	handler := server.API(&apiConfig)

	host := "127.0.0.1:" + strconv.Itoa(c.port)
	// set reasonable defaults here to avoid complexity exposing these as CLI args
	readTimeout := 5 * time.Second
	writeTimeout := 5 * time.Second
	shutdownTimeout := 5 * time.Second

	api := http.Server{
		Addr:         host,
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	log.With("host", host).Info("Starting server")

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
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		// Asking listener to shutdown and load shed.
		err := api.Shutdown(ctx)
		if err != nil {
			log.Infof("main : Graceful shutdown did not complete in %v : %v", shutdownTimeout, err)
			return api.Close()
		}
	}
	return err

}

func openBrowser(url string) error {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	return err
}

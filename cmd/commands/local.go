package commands

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
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

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Exec function for this command.
func (c *LocalCommand) Exec(ctx context.Context, _ []string) error {
	var cfg ServerCommand
	cfg.Host = "127.0.0.1:" + strconv.Itoa(c.port)
	// set reasonable defaults here to avoid complexity exposing these as CLI args
	cfg.ReadTimeout = 5 * time.Second
	cfg.WriteTimeout = 5 * time.Second
	cfg.ShutdownTimeout = 5 * time.Second
	cfg.Demo = c.demo

	// iamzero writes config to ~/.iamzero.ini, to allow developers
	// to set consistent settings between different projects they work on

	// check whether an iamzero config file exists already
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	file := path.Join(home, ".iamzero.ini")

	url := "http://localhost:" + strconv.Itoa(c.port)

	if _, err := os.Stat(file); err == nil {
		c.log("Loading your iamzero config file (" + file + ")")
		// config file exists
		cfgFile, err := ini.Load(file)
		if err != nil {
			return err
		}
		savedToken := cfgFile.Section("iamzero").Key("token")
		savedUrl := cfgFile.Section("iamzero").Key("url")

		if savedUrl.String() != url {
			c.log("The URL in your config file (" + savedUrl.String() + ") was different to the URL your local iamzero server will run on (" + url + "). Updating your config file URL to be " + url + "...")
			savedUrl.SetValue(url)
			err := cfgFile.SaveTo(file)
			if err != nil {
				return err
			}
		}
		cfg.Token = savedToken.String()

	} else if os.IsNotExist(err) {
		// config file does not exist
		c.log(file + " does not exist - initialising new config")
		token, err := randomHex(16)
		if err != nil {
			return err
		}

		cfg.Token = token

		cfgFile := ini.Empty()
		cfgFile.Section("iamzero").Key("token").SetValue(token)
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

	urlWithToken := url + "?token=" + cfg.Token

	err = openBrowser(urlWithToken)
	if err != nil {
		c.log("error opening browser: ", err.Error())
	}

	return cfg.Exec(ctx, nil)
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

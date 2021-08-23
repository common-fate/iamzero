package commands

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/common-fate/iamzero/pkg/applier"
	"github.com/common-fate/iamzero/pkg/storage"
	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// ApplyCommand configuration object
type ApplyCommand struct {
	rootConfig *RootConfig
	out        io.Writer

	logLevel          string
	applierBinaryPath string
}

func askForConfirmation() bool {
	var response string

	_, err := fmt.Scanln(&response)
	if err != nil {
		log.Fatal(err)
	}

	switch strings.ToLower(response) {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	default:
		fmt.Println("Your input doesn't match what we expected, please type (y)es or (n)o and then press enter: ")
		return askForConfirmation()
	}
}

// NewLocNewApplyCommandalCommand creates a new ffcli.Command
func NewApplyCommand(rootConfig *RootConfig, out io.Writer) *ffcli.Command {
	c := ApplyCommand{
		rootConfig: rootConfig,
		out:        out,
	}

	fs := flag.NewFlagSet("iamzero apply", flag.ExitOnError)

	fs.StringVar(&c.logLevel, "log-level", "info", "the log level (must match go.uber.org/zap log levels)")
	fs.StringVar(&c.applierBinaryPath, "applier-binary-path", "iamzero-cdk-applier", "the path to the IAM Zero CDK applier binary")

	rootConfig.RegisterFlags(fs)

	return &ffcli.Command{
		Name:       "apply",
		ShortUsage: "iamzero apply [flags] [<prefix>]",
		ShortHelp:  "Apply IAM Zero findings to your local codebase",
		FlagSet:    fs,
		Options:    []ff.Option{ff.WithEnvVarPrefix("IAMZERO")},
		Exec:       c.Exec,
	}
}

// Exec function for this command.
// The provided argument is the path to the CDK project to scan.
func (c *ApplyCommand) Exec(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("You must provide a path to the project to apply findings to. For example, 'iamzero apply .'")
	}
	projectPath := args[0]

	if c.applierBinaryPath == "" {
		return errors.New("the IAMZERO_APPLIER_BINARY_PATH variable must be set with a path to the IAM Zero CDK applier")
	}

	cfg := zap.NewDevelopmentConfig()
	err := cfg.Level.UnmarshalText([]byte(c.logLevel))
	if err != nil {
		return err
	}
	logProd, err := cfg.Build()
	if err != nil {
		return errors.Wrap(err, "can't initialize zap logger")
	}

	log := logProd.Sugar()

	log.With("projectDir", projectPath).Debug("project dir")

	db, err := storage.OpenBoltDB()
	if err != nil {
		return err
	}

	// if the directory contains a `cdk.json` file, it's a CDK project
	_, err = os.Stat(path.Join(projectPath, "cdk.json"))
	if os.IsNotExist(err) {
		return fmt.Errorf("We couldn't find a CDK project at %s. Please ensure that you are providing a path to a CDK project (which should contain a 'cdk.json' file)", projectPath)
	} else if err != nil {
		return err
	}

	fmt.Println("Synthesizing the CDK stack...")

	cmd := exec.CommandContext(ctx, "cdk", "synth")
	cmd.Dir = projectPath

	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return err
	}

	// After the stack is synthesized the manifest file will be available
	// at {projectDir}/cdk.out/manifest.json
	manifest := path.Join(projectPath, "cdk.out", "manifest.json")
	log.With("manifest", manifest).Debug("Stack synthesized")

	policyStorage := storage.NewBoltPolicyStorage(db)

	findings, err := policyStorage.ListForStatus("active")
	if err != nil {
		return err
	}
	for _, f := range findings {
		if f.CDKFinding != nil && f.CDKFinding.Role.CDKPath != "" {
			findingStr, err := json.Marshal(f.CDKFinding)
			if err != nil {
				return err
			}
			log.With("finding", f.ID).Debug("applying finding")

			cmd := exec.CommandContext(ctx, c.applierBinaryPath, "-f", string(findingStr), "-m", manifest)
			cmd.Stderr = os.Stderr
			stdout, err := cmd.Output()

			if err != nil {
				return err
			}

			var out applier.ApplierOutput

			err = json.Unmarshal(stdout, &out)
			if err != nil {
				return err
			}
			log.With("out", out).Debug("parsed applier output")

			fmt.Printf("[IAM ZERO] We found a recommended change based on our least-privilege policy analysis:\n\n")

			for _, o := range out {
				diff, err := applier.GetDiff(o.Path, o.Contents)
				if err != nil {
					return err
				}
				fmt.Println(diff)

			}
			fmt.Printf("[IAM ZERO] Accept the change? [y/n]: ")

			confim := askForConfirmation()

			if confim {
				for _, o := range out {
					err = ioutil.WriteFile(o.Path, []byte(o.Contents), 0644)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

package commands

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"

	"github.com/common-fate/iamzero/pkg/applier"
	cdkApplier "github.com/common-fate/iamzero/pkg/applier/cdk"
	terraformApplier "github.com/common-fate/iamzero/pkg/applier/terraform"
	"github.com/common-fate/iamzero/pkg/recommendations"
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
	skipSynth         bool
}

func promptForConfirmation() bool {
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
		return promptForConfirmation()
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
	fs.BoolVar(&c.skipSynth, "skip-synth", false, "skip running the 'cdk synth' command as part of the analysis")

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

func renderProjectDetectedMessage(name string, projectPath string) error {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return err
	}
	fmt.Printf("We detected a %s project at %s\n", name, absPath)
	return nil
}

func fetchEnabledAcionsForPolicy(actionStorage *storage.BoltActionStorage, policyID string) ([]recommendations.AWSAction, error) {
	actions, err := actionStorage.ListForPolicy(policyID)
	if err != nil {
		return nil, err
	}
	var enabledActions []recommendations.AWSAction
	for _, a := range actions {
		if a.Enabled {
			enabledActions = append(enabledActions, a)
		}
	}
	return enabledActions, nil
}
func promptForChangeAcceptance() bool {
	fmt.Printf("[IAM ZERO] Accept the change? [y/n]: ")
	return promptForConfirmation()
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

	policyStorage := storage.NewBoltPolicyStorage(db)
	actionStorage := storage.NewBoltActionStorage(db)
	findings, err := policyStorage.ListForStatus("active")
	if err != nil {
		return err
	}

	// Here we can instansiate all our appliers
	appliers := applier.PolicyAppliers{
		terraformApplier.TerraformIAMPolicyApplier{applier.AWSIAMPolicyApplier{
			ProjectPath: projectPath, Logger: log},
			nil, nil},
		cdkApplier.CDKIAMPolicyApplier{applier.AWSIAMPolicyApplier{
			ProjectPath: projectPath, Logger: log},
			nil, c.skipSynth, ctx, c.applierBinaryPath, ""},
	}
	// if the directory contains a `cdk.json` file, it's a CDK project
	// if the directory contains a `main.tf` file, it's a Terraform project
	projectDetected := false
	for _, applier := range appliers {
		if applier.Detect() {
			if err := renderProjectDetectedMessage(applier.GetProjectName(), ""); err != nil {
				return err
			}
			applier.Init()
			projectDetected = true
			for _, policy := range findings {
				actions, err := fetchEnabledAcionsForPolicy(actionStorage, policy.ID)
				if err != nil {
					return err
				}
				plan, err := applier.Plan(&policy, actions)
				if err != nil {
					return err
				}
				fmt.Printf("\n💡 We found a recommended change based on our least-privilege policy analysis:\n\n")
				plan.RenderDiff()
				if promptForChangeAcceptance() {
					applier.Apply(plan)
				}
			}
		}
	}
	if !projectDetected {
		return fmt.Errorf("we couldn't find a CDK project or a Terraform Project at %s. Please ensure that you are providing a path to a CDK project (which should contain a 'cdk.json' file) or a Terraform project (which should contain a 'main.tf' file)", projectPath)
	}

	return nil
}

package commands

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	collectorApp "github.com/common-fate/iamzero/cmd/collector/app"
	consoleApp "github.com/common-fate/iamzero/cmd/console/app"
	"go.uber.org/zap"

	"github.com/common-fate/iamzero/pkg/audit"
	"github.com/common-fate/iamzero/pkg/cloudtrail"
	"github.com/peterbourgon/ff/v3/ffcli"
)

// ScanCommand configuration object
type ScanCommand struct {
	rootConfig *RootConfig
	out        io.Writer

	Collector *collectorApp.Collector
	Console   *consoleApp.Console
	Auditor   *audit.Auditor

	roleName               string
	logLevel               string
	athenaCloudTrailBucket string
	athenaResultsLocation  string
	account                string
}

// NewScanCommand creates a new ffcli.Command
func NewScanCommand(rootConfig *RootConfig, out io.Writer) *ffcli.Command {
	c := ScanCommand{
		rootConfig: rootConfig,
		out:        out,
	}

	c.Collector = collectorApp.New()
	c.Console = consoleApp.New()
	c.Auditor = audit.New()

	fs := flag.NewFlagSet("iamzero scan", flag.ExitOnError)

	// register CLI flags for other components
	c.Collector.AddFlags(fs)
	c.Console.AddFlags(fs)
	c.Auditor.AddFlags(fs)

	fs.StringVar(&c.logLevel, "log-level", "info", "the log level (must match go.uber.org/zap log levels)")
	fs.StringVar(&c.roleName, "role", "", "the name of the role to query events for")
	fs.StringVar(&c.athenaCloudTrailBucket, "cloudtrail-bucket", "", "the S3 bucket that CloudTrail logs are stored in")
	fs.StringVar(&c.athenaResultsLocation, "results-location", "", "the S3 path to store Athena query results in")
	fs.StringVar(&c.account, "account", "", "the AWS account to query CloudTrail logs for")

	rootConfig.RegisterFlags(fs)

	return &ffcli.Command{
		Name:       "scan",
		ShortUsage: "iamzero scan [flags] [<prefix>]",
		ShortHelp:  "Query cloud audit trails for events",
		FlagSet:    fs,
		Exec:       c.Exec,
	}
}

// Exec function for this command.
func (c *ScanCommand) Exec(ctx context.Context, args []string) error {
	cfg := zap.NewDevelopmentConfig()
	err := cfg.Level.UnmarshalText([]byte(c.logLevel))
	if err != nil {
		return err
	}
	logProd, err := cfg.Build()
	if err != nil {
		return err
	}
	log := logProd.Sugar()

	if c.roleName == "" {
		return errors.New("the -role argument must be provided")
	}

	if c.athenaCloudTrailBucket == "" {
		return errors.New("the -cloudtrail-bucket argument must be provided")
	}
	if c.athenaResultsLocation == "" {
		return errors.New("the -results-location argument must be provided")
	}
	if c.account == "" {
		return errors.New("the -account argument must be provided")
	}

	fmt.Printf("Querying CloudTrail logs for %s\n", c.roleName)

	a := cloudtrail.NewCloudTrailAuditor(&cloudtrail.CloudTrailAuditorParams{
		Log:                    log,
		AthenaCloudTrailBucket: c.athenaCloudTrailBucket,
		AthenaOutputLocation:   c.athenaResultsLocation,
	})
	err = a.GetActionsForRole(ctx, c.account, c.roleName)
	if err != nil {
		return err
	}

	return nil
}

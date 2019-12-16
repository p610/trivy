package config

import (
	"os"
	"strings"
	"time"

	"github.com/genuinetools/reg/registry"
	"github.com/urfave/cli"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

	dbTypes "github.com/aquasecurity/trivy-db/pkg/types"
	"github.com/aquasecurity/trivy/pkg/log"
)

type Config struct {
	context *cli.Context
	logger  *zap.SugaredLogger

	Quiet      bool
	NoProgress bool
	Debug      bool

	CacheDir       string
	Reset          bool
	DownloadDBOnly bool
	SkipUpdate     bool
	ClearCache     bool

	Input    string
	output   string
	Format   string
	Template string

	Timeout       time.Duration
	vulnType      string
	Light         bool
	severities    string
	IgnoreFile    string
	IgnoreUnfixed bool
	ExitCode      int

	// these variables are generated by Init()
	ImageName  string
	VulnType   []string
	Output     *os.File
	Severities []dbTypes.Severity
	AppVersion string

	// deprecated
	onlyUpdate string
	// deprecated
	refresh bool
	// deprecated
	autoRefresh bool
}

func New(c *cli.Context) (Config, error) {
	debug := c.Bool("debug")
	quiet := c.Bool("quiet")
	logger, err := log.NewLogger(debug, quiet)
	if err != nil {
		return Config{}, xerrors.New("failed to create a logger")
	}
	return Config{
		context: c,
		logger:  logger,

		Quiet:      quiet,
		NoProgress: c.Bool("no-progress"),
		Debug:      debug,

		CacheDir:       c.String("cache-dir"),
		Reset:          c.Bool("reset"),
		DownloadDBOnly: c.Bool("download-db-only"),
		SkipUpdate:     c.Bool("skip-update"),
		ClearCache:     c.Bool("clear-cache"),

		Input:    c.String("input"),
		output:   c.String("output"),
		Format:   c.String("format"),
		Template: c.String("template"),

		Timeout:       c.Duration("timeout"),
		vulnType:      c.String("vuln-type"),
		Light:         c.Bool("light"),
		severities:    c.String("severity"),
		IgnoreFile:    c.String("ignorefile"),
		IgnoreUnfixed: c.Bool("ignore-unfixed"),
		ExitCode:      c.Int("exit-code"),

		onlyUpdate:  c.String("only-update"),
		refresh:     c.Bool("refresh"),
		autoRefresh: c.Bool("auto-refresh"),
	}, nil
}

func (c *Config) Init() (err error) {
	if c.onlyUpdate != "" || c.refresh || c.autoRefresh {
		c.logger.Warn("--only-update, --refresh and --auto-refresh are unnecessary and ignored now. These commands will be removed in the next version.")
	}
	if c.SkipUpdate && c.DownloadDBOnly {
		return xerrors.New("The --skip-update and --download-db-only option can not be specified both")
	}

	c.Severities = c.splitSeverity(c.severities)
	c.VulnType = strings.Split(c.vulnType, ",")
	c.AppVersion = c.context.App.Version

	// --clear-cache, --download-db-only and --reset don't conduct the scan
	if c.ClearCache || c.DownloadDBOnly || c.Reset {
		return nil
	}

	args := c.context.Args()
	if c.Input == "" && len(args) == 0 {
		c.logger.Error(`trivy requires at least 1 argument or --input option`)
		cli.ShowAppHelp(c.context)
		return xerrors.New("arguments error")
	}

	c.Output = os.Stdout
	if c.output != "" {
		if c.Output, err = os.Create(c.output); err != nil {
			return xerrors.Errorf("failed to create an output file: %w", err)
		}
	}

	if c.Input == "" {
		c.ImageName = args[0]
	}

	// Check whether 'latest' tag is used
	if c.ImageName != "" {
		image, err := registry.ParseImage(c.ImageName)
		if err != nil {
			return xerrors.Errorf("invalid image: %w", err)
		}
		if image.Tag == "latest" {
			c.logger.Warn("You should avoid using the :latest tag as it is cached. You need to specify '--clear-cache' option when :latest image is changed")
		}
	}

	return nil
}

func (c *Config) splitSeverity(severity string) []dbTypes.Severity {
	c.logger.Debugf("Severities: %s", severity)
	var severities []dbTypes.Severity
	for _, s := range strings.Split(severity, ",") {
		severity, err := dbTypes.NewSeverity(s)
		if err != nil {
			c.logger.Warnf("unknown severity option: %s", err)
		}
		severities = append(severities, severity)
	}
	return severities
}
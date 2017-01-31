package cmd

import (
	"flag"
	"fmt"
	"io"
	"log"
	"reflect"
	"sort"

	"strings"
	"time"

	"github.com/openebs/mtest/config"
	"github.com/openebs/mtest/util"
	"github.com/openebs/mtest/util/flag-helpers"
	"github.com/openebs/mtest/util/gated-writer"

	"github.com/hashicorp/go-syslog"
	"github.com/hashicorp/logutils"
	"github.com/mitchellh/cli"
)

// gracefulTimeout controls how long we wait before forcefully terminating
const gracefulTimeout = 5 * time.Second

// EBSCommand is a cli implementation that executes EBS APIs against
// Maya server. In other words this a single command that runs the
// entire Mayaserver compatible EBS test suite. This should take care
// of EBS zone, credential management, metadata & other non-functional
// requirements. In addition, EBS version compatibility testing should
// also be taken care of by this single command line implementation.
type EBSCommand struct {
	Ui        cli.Ui
	args      []string
	logFilter *logutils.LevelFilter
	logOutput io.Writer
}

func (c *EBSCommand) readMtestConfig() *config.MtestConfig {
	var configPath []string

	// Make a new, empty mtest config.
	cmdConfig := &config.MtestConfig{}

	flags := flag.NewFlagSet("ebs", flag.ContinueOnError)
	flags.Usage = func() { c.Ui.Error(c.Help()) }

	// options
	flags.Var((*flaghelper.StringFlag)(&configPath), "config", "config")

	if err := flags.Parse(c.args); err != nil {
		return nil
	}

	// Load the configuration
	mtconfig := config.DefaultMtestConfig()

	for _, path := range configPath {
		current, err := config.LoadMtestConfig(path)
		if err != nil {
			c.Ui.Error(fmt.Sprintf(
				"Error loading configuration from %s: %s", path, err))
			return nil
		}

		// The user asked us to load some config here but we didn't find any,
		// so we'll complain but continue.
		if current == nil || reflect.DeepEqual(current, &config.MtestConfig{}) {
			c.Ui.Warn(fmt.Sprintf("No configuration loaded from %s", path))
		}

		if mtconfig == nil {
			mtconfig = current
		} else {
			mtconfig = mtconfig.Merge(current)
		}
	}

	// Merge any CLI options over config file options
	mtconfig = mtconfig.Merge(cmdConfig)

	return mtconfig
}

// setupLoggers is used to setup the logGate, logWriter, and our logOutput
func (c *EBSCommand) setupLoggers(mtconfig *config.MtestConfig) (*gatedwriter.Writer, *util.LogWriter, io.Writer) {
	// Setup logging. First create the gated log writer, which will
	// store logs until we're ready to show them. Then create the level
	// filter, filtering logs of the specified level.
	logGate := &gatedwriter.Writer{
		Writer: &cli.UiWriter{Ui: c.Ui},
	}

	c.logFilter = util.LevelFilter()
	c.logFilter.MinLevel = logutils.LogLevel(strings.ToUpper(mtconfig.LogLevel))
	c.logFilter.Writer = logGate
	if !util.ValidateLevelFilter(c.logFilter.MinLevel, c.logFilter) {
		c.Ui.Error(fmt.Sprintf(
			"Invalid log level: %s. Valid log levels are: %v",
			c.logFilter.MinLevel, c.logFilter.Levels))
		return nil, nil, nil
	}

	// Check if syslog is enabled
	var syslog io.Writer
	if mtconfig.EnableSyslog {
		l, err := gsyslog.NewLogger(gsyslog.LOG_NOTICE, mtconfig.SyslogFacility, "mtest")
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Syslog setup failed: %v", err))
			return nil, nil, nil
		}
		syslog = &util.SyslogWrapper{l, c.logFilter}
	}

	// Create a log writer, and wrap a logOutput around it
	logWriter := util.NewLogWriter(512)
	var logOutput io.Writer
	if syslog != nil {
		logOutput = io.MultiWriter(c.logFilter, logWriter, syslog)
	} else {
		logOutput = io.MultiWriter(c.logFilter, logWriter)
	}
	c.logOutput = logOutput
	log.SetOutput(logOutput)
	return logGate, logWriter, logOutput
}

func (c *EBSCommand) Run(args []string) int {
	c.Ui = &cli.PrefixedUi{
		OutputPrefix: "==> ",
		InfoPrefix:   "    ",
		ErrorPrefix:  "==> ",
		Ui:           c.Ui,
	}

	// Parse our configs
	c.args = args
	mtconfig := c.readMtestConfig()
	if mtconfig == nil {
		return 1
	}

	// Setup the log outputs
	//logGate, _, logOutput := c.setupLoggers(mtconfig)
	logGate, _, _ := c.setupLoggers(mtconfig)
	if logGate == nil {
		return 1
	}

	// Log config files
	if len(mtconfig.Files) > 0 {
		c.Ui.Info(fmt.Sprintf("Loaded configuration from %s", strings.Join(mtconfig.Files, ", ")))
	} else {
		c.Ui.Info("No configuration files loaded")
	}

	// Compile Maya server information for output later
	info := make(map[string]string)
	info["version"] = fmt.Sprintf("%s%s", mtconfig.Version, mtconfig.VersionPrerelease)
	info["log level"] = mtconfig.LogLevel

	// Sort the keys for output
	infoKeys := make([]string, 0, len(info))
	for key := range info {
		infoKeys = append(infoKeys, key)
	}
	sort.Strings(infoKeys)

	// Mtest configuration output
	padding := 18
	c.Ui.Output("Mtest configuration:\n")
	for _, k := range infoKeys {
		c.Ui.Info(fmt.Sprintf(
			"%s%s: %s",
			strings.Repeat(" ", padding-len(k)),
			strings.Title(k),
			info[k]))
	}
	c.Ui.Output("")

	// Output the header that the server has started
	c.Ui.Output("Mtest ebs run started! Log data will stream in below:\n")

	// Enable log streaming
	logGate.Flush()

	err := 0

	return err
}

func (c *EBSCommand) Synopsis() string {
	return "Runs Mtest ebs testsuite"
}

func (c *EBSCommand) Help() string {
	helpText := `
Usage: mtest ebs [options]

  Runs Mtest ebs testsuite.

  The Mtest's configuration primarily comes from the config
  files used. The config file path can be passed as a CLI argument,
  listed below.

General Options :
  
  -config=<path>
    The path to either a single config file or a directory of config
    files to use for configuring Mtest. This option may be
    specified multiple times. If multiple config files are used, the
    values from each will be merged together. During merging, values
    from files found later in the list are merged over values from
    previously parsed files.
 `
	return strings.TrimSpace(helpText)
}

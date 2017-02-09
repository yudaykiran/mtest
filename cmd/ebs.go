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
	"github.com/openebs/mtest/logging"
	"github.com/openebs/mtest/logging/flag-helpers"
	"github.com/openebs/mtest/logging/gated-writer"
	"github.com/openebs/mtest/mtest"

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
	Ui          cli.Ui
	args        []string
	logFilter   *logutils.LevelFilter
	multiLogger io.Writer
}

// This will provide a final validated mtest configuration.
func (c *EBSCommand) readMtestConfig() *config.MtestConfig {
	// Variable to hold a set of config paths
	var configPaths []string

	// Make a new, empty mtest config.
	// TODO
	//    when user will provide config options
	//    while using this CLI
	cmdConfig := &config.MtestConfig{}

	flags := flag.NewFlagSet("ebs", flag.ContinueOnError)
	flags.Usage = func() { c.Ui.Error(c.Help()) }

	// options
	flags.Var((*flaghelper.StringFlag)(&configPaths), "config", "path(s) of config file(s)")

	err := flags.Parse(c.args)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading configuration from %v. %s", configPaths, err))

		return nil
	}

	// Get the default configuration
	mtconfig := config.DefaultMtestConfig()

	for _, path := range configPaths {
		current, err := config.LoadMtestConfig(path)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error loading configuration from %s: %s", path, err))

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

// setupLoggers is used to setup various logger variants named as
// gatedLogger, logPipe, and multiLogger
func (c *EBSCommand) setupLoggers(mtconfig *config.MtestConfig) (*gatedwriter.Writer, *logging.LogWriter, io.Writer) {
	// Setup filtered-gated-logger !!
	// First create the gated log writer, which will buffer logs until we're ready.
	gatedLogger := &gatedwriter.Writer{
		// Set gated logger's writer against that of CLI's
		// Any messages to this CLI will be logged as per
		// gated logger's logic
		Writer: &cli.UiWriter{Ui: c.Ui},
	}

	// Now create the level filter, filtering logs of the specified level.
	// This filtering will be set on top of just created gated logger
	c.logFilter = logging.LevelFilter()
	c.logFilter.MinLevel = logutils.LogLevel(strings.ToUpper(mtconfig.LogLevel))
	c.logFilter.Writer = gatedLogger

	if !logging.ValidateLevelFilter(c.logFilter.MinLevel, c.logFilter) {
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

		// Setup syslog-filtered-gated-logger !!
		syslog = &logging.SyslogWrapper{l, c.logFilter}
	}

	// Create a log pipe too
	// This acts as a log sink with pipes (i.e. otherwise known as handles)
	// that can be directed to various sink handlers.
	//    NOTE:
	//      These handlers can be registered (& de-registered) at a later
	//      point of time dynamically.
	logPipe := logging.NewLogWriter(512)

	// Create a multi-write logger around all the above log variants
	var multiLogger io.Writer
	if syslog != nil {
		multiLogger = io.MultiWriter(c.logFilter, logPipe, syslog)
	} else {
		multiLogger = io.MultiWriter(c.logFilter, logPipe)
	}

	c.multiLogger = multiLogger

	// Set our multi-write logger as logging mechanism
	// Why ? Coz our multi-write logger has everything !!
	log.SetOutput(multiLogger)

	// Provide all the logger variants
	return gatedLogger, logPipe, multiLogger
}

func (c *EBSCommand) Run(args []string) int {
	// Decorate this CLI's UI
	c.Ui = &cli.PrefixedUi{
		OutputPrefix: " ",
		InfoPrefix:   "INFO:  ",
		ErrorPrefix:  "ERROR: ",
		Ui:           c.Ui,
	}

	// Set the args which may have the mtest config
	c.args = args
	// Parse the mtest config
	mtconfig := c.readMtestConfig()
	if mtconfig == nil {
		return 1
	}

	// Setup the log outputs
	gatedLogger, _, multiLogger := c.setupLoggers(mtconfig)
	if gatedLogger == nil {
		return 1
	}
	defer gatedLogger.Flush()

	if multiLogger == nil {
		c.Ui.Error(fmt.Sprintf("Log setup failed. Nil instance of multi-write logger found."))
		return 1
	}

	// Inform about mt configuration file(s)
	if len(mtconfig.Files) > 0 {
		c.Ui.Info(fmt.Sprintf("Loaded configuration from %s", strings.Join(mtconfig.Files, ", ")))
	} else {
		c.Ui.Info("No configuration files loaded")
	}

	// Build Mtest information for messaging
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
	c.Ui.Output("Mtest ebs run started! Log data will start streaming:\n")

	// ebs cli is meant to run Maya Server
	// Get a Mtest instance that is associated with Maya Server
	//
	// NOTE:
	//  We will use the multi-write logger
	mt, err := mtest.NewMserverRunner(multiLogger)
	if err != nil {
		c.Ui.Error(err.Error())
		return 0
	}

	// Start EBS use cases
	_, err = mt.Start()
	if err != nil {
		c.Ui.Error(err.Error())
		return 0
	}

	return 0
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

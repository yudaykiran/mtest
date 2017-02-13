package cmd

import (
	"flag"
	"fmt"
	"sort"

	"strings"
	"time"

	"github.com/openebs/mtest/config"
	"github.com/openebs/mtest/logging"
	"github.com/openebs/mtest/mtest"

	"github.com/hashicorp/logutils"
	"github.com/mitchellh/cli"
	"github.com/openebs/mtest/logging/flag-helpers"
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
	mtConfMake  config.MtestConfigMaker
	wtrVarsMake logging.WriterVariantsMaker
	mtestMake   mtest.MtestMaker
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

	if c.mtConfMake == nil {
		c.Ui.Error(fmt.Sprintf("Mtest-config-maker instance is nil"))
		return nil
	}

	mtconfig, err := c.mtConfMake.Make(configPaths)
	if err != nil {
		c.Ui.Error(err.Error())
		return nil
	}

	// Merge any CLI options over config file options
	mtconfig = mtconfig.Merge(cmdConfig)

	return mtconfig
}

func (c *EBSCommand) SetMtestMaker(mtestMake mtest.MtestMaker) {
	c.mtestMake = mtestMake
}

func (c *EBSCommand) SetMtConfMaker(mtConfMake config.MtestConfigMaker) {
	c.mtConfMake = mtConfMake
}

func (c *EBSCommand) SetWriterMaker(wtrVarsMake logging.WriterVariantsMaker) {
	c.wtrVarsMake = wtrVarsMake
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

	if c.wtrVarsMake == nil {
		c.Ui.Error(fmt.Sprintf("Writer-variants-maker instance is nil."))
		return 1
	}

	// Setup the log outputs
	gatedLogger := c.wtrVarsMake.GatedWriter()

	if gatedLogger == nil {
		return 1
	}
	defer gatedLogger.Flush()

	multiLogger := c.wtrVarsMake.MultiWriter()
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

	if c.mtestMake == nil {
		c.Ui.Error(fmt.Sprintf("Mtest-maker instance is nil."))
		return 1
	}

	// ebs cli is meant to run Maya Server
	// Get a Mtest instance that is associated with Maya Server
	mt, err := c.mtestMake.Make()
	if err != nil {
		c.Ui.Error(err.Error())
		return 0
	}

	// Output the header that the server has started
	c.Ui.Output("Mtest ebs run started! Log data will start streaming:\n")

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

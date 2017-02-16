package cmd

import (
	"flag"
	"fmt"
	"sort"
	"sync"

	"strings"
	"time"

	"github.com/openebs/mtest/config"
	"github.com/openebs/mtest/logging"
	"github.com/openebs/mtest/mtest"

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
	m    sync.Mutex
	Ui   cli.Ui
	args []string

	// A flag indicating if dependencies & their initialization
	// have been invoked or not
	initialized bool

	// A dependency that aligns to MtestConfigMaker interface
	mtConfMake config.MtestConfigMaker

	// A dependency that aligns to WriterVariantsMaker interface
	wtrVarsMake logging.WriterVariantsMaker

	// A dependency that aligns to MtestMaker interface
	mtestMake mtest.MtestMaker
}

// IsInitialized indicates if EBSCommand is being
// initialized now
func (c *EBSCommand) IsInitialized() bool {
	c.m.Lock()
	defer c.m.Unlock()

	return c.isInitialized()
}

// isInitialized helper method defines whether EBSCommand
// is being initialized right now
func (c *EBSCommand) isInitialized() bool {
	return c.initialized
}

// SetAll injects various dependencies required for
// EBSCommand's functioning.
// Each of these dependencies align to *Maker interface
//
// NOTE:
//  Once injected, each dependant's Make() function
//  might be executed.
func (c *EBSCommand) SetAll() error {
	c.m.Lock()
	defer c.m.Unlock()

	// Initialize only if not initialized earlier
	if !c.isInitialized() {

		// Build a config maker instance
		if c.mtConfMake == nil {
			c.mtConfMake = &config.MtestConfigMake{}
		}

		// Get the mtest config which will use the config maker
		// instance. This is done so early as config is required
		// for every other thing
		mtconfig := c.readMtestConfig()
		if mtconfig == nil {
			return fmt.Errorf("Could not create mtest config")
		}

		// Build a writer variants maker instance
		if c.wtrVarsMake == nil {
			c.wtrVarsMake = &logging.WriterVariantsMake{
				Existing: &cli.UiWriter{Ui: c.Ui},
			}
		}

		// This will create variants of writer instances
		err := c.wtrVarsMake.Make(strings.ToUpper(mtconfig.LogLevel), mtconfig.EnableSyslog)
		if err != nil {
			return err
		}

		// Build a Mtest maker instance that is associated
		// with MserverRunner
		if c.mtestMake == nil {
			c.mtestMake, err = mtest.NewMserverRunMaker(c.wtrVarsMake.MultiWriter())
			if err != nil {
				return err
			}
		}

		// Once all the above maker instances are set as dependencies
		c.initialized = true
	}

	return nil
}

// This will read & load the Mtest config from the provided paths
// do necessary validations & merge any user privided config
// properties. All of these are done on top of default mtest config.
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

	// Inform about mt configuration file(s)
	if len(mtconfig.Files) > 0 {
		c.Ui.Info(fmt.Sprintf("Loaded configuration from %s", strings.Join(mtconfig.Files, ", ")))
	} else {
		c.Ui.Info("No configuration files loaded")
	}

	// Build Mtest information for messaging
	info := make(map[string]string)
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

	return mtconfig
}

// This CLI sub-command's entry point
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

	// Dependency Injection
	if !c.IsInitialized() {
		err := c.SetAll()
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}
	}

	if c.wtrVarsMake == nil {
		c.Ui.Error(fmt.Sprintf("Writer-variants-maker instance is nil."))
		return 1
	}

	// Defer flush the gatedWriter which is linked to this
	// CLI's io.writer during Dependency Injection
	gatedLogger := c.wtrVarsMake.GatedWriter()
	if gatedLogger == nil {
		return 1
	}
	defer gatedLogger.Flush()

	if c.mtestMake == nil {
		c.Ui.Error(fmt.Sprintf("Mtest-maker instance is nil."))
		return 1
	}

	// ebs cli is meant to run Maya Server
	// Get a Mtest instance that is associated with Maya Server
	mt, err := c.mtestMake.Make()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Output the header that the server has started
	c.Ui.Output("Mtest ebs run started! Log data will start streaming:\n")

	// Start EBS use cases
	rpts, err := mt.Start()
	defer mt.Stop()

	if err != nil {
		c.Ui.Error(err.Error())
		// Exit code is set to 0 as this has nothing to do
		// with running of CLI. CLI execution was fine.
		return 0
	}

	c.Ui.Info(fmt.Sprintf("%+s", rpts))

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

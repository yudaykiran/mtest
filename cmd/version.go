package cmd

import (
	"bytes"
	"fmt"

	"github.com/mitchellh/cli"
)

// VersionCommand is a cli implementation that prints the version.
type VersionCommand struct {
	Revision          string
	Version           string
	VersionPrerelease string
	Ui                cli.Ui
}

func (c *VersionCommand) Help() string {
	return ""
}

func (c *VersionCommand) Run(_ []string) int {
	var versionString bytes.Buffer

	fmt.Fprintf(&versionString, "Mtest v%s", c.Version)
	if c.VersionPrerelease != "" {
		fmt.Fprintf(&versionString, "-%s", c.VersionPrerelease)

		if c.Revision != "" {
			fmt.Fprintf(&versionString, " (%s)", c.Revision)
		}
	}

	c.Ui.Output(versionString.String())
	return 0
}

func (c *VersionCommand) Synopsis() string {
	return "Prints the Mtest version"
}

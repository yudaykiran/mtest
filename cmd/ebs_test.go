package cmd

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
)

func TestCommand_Implements(t *testing.T) {
	var _ cli.Command = &EBSCommand{}
}

func TestCommand_Args(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "mtest")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	type tcase struct {
		args   []string
		errOut string
	}
	tcases := []tcase{
		{
			[]string{},
			"",
		},
	}
	for _, tc := range tcases {
		// Make a new command. We pre-emptively close the shutdownCh
		// so that the command exits immediately instead of blocking.
		ui := new(cli.MockUi)
		shutdownCh := make(chan struct{})
		close(shutdownCh)
		cmd := &EBSCommand{
			Ui: ui,
		}

		if code := cmd.Run(tc.args); code != 0 {
			t.Fatalf("args: %v\nexit: %d\n", tc.args, code)
		}

		if expect := tc.errOut; expect != "" {
			out := ui.ErrorWriter.String()
			if !strings.Contains(out, expect) {
				t.Fatalf("expect to find %q\n\n%s", expect, out)
			}
		}
	}
}

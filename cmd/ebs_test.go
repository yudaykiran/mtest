package cmd

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/openebs/mtest/config"
	"github.com/openebs/mtest/mtest"
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
		name       string
		args       []string
		exit       int
		out        string
		mtConfMake config.MtestConfigMaker
		mtestMake  mtest.MtestMaker
	}

  // List down all the permutations & combinations to
  // achieve maximum coverage
	tcases := []tcase{
		{
			"Negative Test Case: 1",
			[]string{},
			1,
			"Mtest-config-maker instance is nil",
			nil,
			nil,
		},
		{
			"Negative Test Case: 2",
			[]string{"-config=" + tmpDir},
			1,
			"No configuration loaded",
			&config.MtestConfigMake{},
			nil,
		},
	}

	for _, tc := range tcases {
		ui := new(cli.MockUi)

		cmd := &EBSCommand{
			Ui:         ui,
			mtConfMake: tc.mtConfMake,
			mtestMake:  tc.mtestMake,
		}

		// Run the cmd with test case' args
		code := cmd.Run(tc.args)
		expectMsg := tc.out
		actualMsg := ui.ErrorWriter.String()

		if code != tc.exit {
			t.Fatalf("\t\nerror: Exit code mismatch, \t\nTest Name: '%s', \t\nexpected: '0', \t\ngot: '%d', \t\nmsg: '%v'", tc.name, code, actualMsg)
		}

		if expectMsg == "" && actualMsg != "" {
			t.Fatalf("\t\nerror: Message mismatch, \t\nTest Name: '%s', \t\nexpected: '', \t\nactual: '%s'", tc.name, actualMsg)
		}

		if !strings.Contains(actualMsg, expectMsg) {
			t.Fatalf("\t\nerror: Message mismatch, \t\nTest Name: '%s', \t\nexpected: '%s', \t\nactual: '%s'", tc.name, expectMsg, actualMsg)
		}
	}
}

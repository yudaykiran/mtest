package config

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestMayaConfig_Parse(t *testing.T) {
	cases := []struct {
		File   string
		Result *MtestConfig
		Err    bool
	}{
		{
			"dummy_mtest_config.hcl",
			&MtestConfig{
				LogLevel:       "INFO",
				EnableSyslog:   true,
				SyslogFacility: "LOCAL1",
			},
			false,
		},
	}

	for _, tc := range cases {
		t.Logf("Testing parse: %s", tc.File)

		path, err := filepath.Abs(filepath.Join("../mockit/", tc.File))
		if err != nil {
			t.Fatalf("filepath err: %s\n\n%s", tc.File, err)
			continue
		}

		actual, err := ParseMtestConfigFile(path)
		if (err != nil) != tc.Err {
			t.Fatalf("fileparse err: %s\n\n%s", tc.File, err)
			continue
		}

		if !reflect.DeepEqual(actual, tc.Result) {
			t.Fatalf("fileequal err: %s\nactual:\t\t%q\n\nexpected:\t%q", tc.File, actual, tc.Result)
		}
	}
}

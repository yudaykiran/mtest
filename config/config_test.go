package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var (
	// trueValue/falseValue are used to get a pointer to a boolean
	trueValue  = true
	falseValue = false
)

func TestMtestConfig_Merge(t *testing.T) {
	c1 := &MtestConfig{
		LogLevel:       "INFO",
		EnableSyslog:   false,
		SyslogFacility: "local0.info",
	}

	c2 := &MtestConfig{
		LogLevel:       "DEBUG",
		EnableSyslog:   true,
		SyslogFacility: "local0.debug",
	}

	result := c1.Merge(c2)
	if !reflect.DeepEqual(result, c2) {
		t.Fatalf("bad:\n%#v\n%#v", result, c2)
	}
}

func TestParseMtestConfigFile(t *testing.T) {
	// Fails if the file doesn't exist
	if _, err := ParseMtestConfigFile("/unicorns/leprechauns"); err == nil {
		t.Fatalf("expected error, got nothing")
	}

	fh, err := ioutil.TempFile("", "mtest")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(fh.Name())

	// Invalid content returns error
	if _, err := fh.WriteString("nope;!!!"); err != nil {
		t.Fatalf("err: %s", err)
	}
	if _, err := ParseMtestConfigFile(fh.Name()); err == nil {
		t.Fatalf("expected load error, got nothing")
	}

	// Valid content parses successfully
	if err := fh.Truncate(0); err != nil {
		t.Fatalf("err: %s", err)
	}
	if _, err := fh.Seek(0, 0); err != nil {
		t.Fatalf("err: %s", err)
	}
	if _, err := fh.WriteString(`{"log_level":"INFO"}`); err != nil {
		t.Fatalf("err: %s", err)
	}

	config, err := ParseMtestConfigFile(fh.Name())
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if config.LogLevel != "INFO" {
		t.Fatalf("bad log level: expected: INFO, got: %q", config.LogLevel)
	}
}

func TestLoadMtestConfigDir(t *testing.T) {
	// Fails if the dir doesn't exist.
	if _, err := LoadMtestConfigDir("/unicorns/leprechauns"); err == nil {
		t.Fatalf("expected error, got nothing")
	}

	dir, err := ioutil.TempDir("", "mtest")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(dir)

	// Returns empty config on empty dir
	config, err := LoadMtestConfig(dir)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if config == nil {
		t.Fatalf("should not be nil")
	}

	file1 := filepath.Join(dir, "conf1.hcl")
	err = ioutil.WriteFile(file1, []byte(`{"log_level":"debug"}`), 0600)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	file2 := filepath.Join(dir, "conf2.hcl")
	err = ioutil.WriteFile(file2, []byte(`{"enable_syslog":"true"}`), 0600)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	file3 := filepath.Join(dir, "conf3.hcl")
	err = ioutil.WriteFile(file3, []byte(`nope;!!!`), 0600)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Fails if we have a bad config file
	if _, err := LoadMtestConfigDir(dir); err == nil {
		t.Fatalf("expected load error, got nothing")
	}

	if err := os.Remove(file3); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Works if configs are valid
	config, err = LoadMtestConfigDir(dir)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if config.LogLevel != "debug" || config.EnableSyslog != true {
		t.Fatalf("bad: %#v", config)
	}
}

func TestConfig_LoadMayaConfig(t *testing.T) {
	// Fails if the target doesn't exist
	if _, err := LoadMtestConfig("/unicorns/leprechauns"); err == nil {
		t.Fatalf("expected error, got nothing")
	}

	fh, err := ioutil.TempFile("", "mtest")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Remove(fh.Name())

	if _, err := fh.WriteString(`{"enable_syslog":"true"}`); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Works on a config file
	config, err := LoadMtestConfig(fh.Name())
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if config.EnableSyslog != true {
		t.Fatalf("bad: %#v", config)
	}

	expectedConfigFiles := []string{fh.Name()}
	if !reflect.DeepEqual(config.Files, expectedConfigFiles) {
		t.Errorf("Loaded configs don't match\nExpected\n%+vGot\n%+v\n",
			expectedConfigFiles, config.Files)
	}

	dir, err := ioutil.TempDir("", "mtest")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(dir)

	file1 := filepath.Join(dir, "config1.hcl")
	err = ioutil.WriteFile(file1, []byte(`{"log_level":"info"}`), 0600)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Works on config dir
	config, err = LoadMtestConfig(dir)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if config.LogLevel != "info" {
		t.Fatalf("bad: %#v", config)
	}

	expectedConfigFiles = []string{file1}
	if !reflect.DeepEqual(config.Files, expectedConfigFiles) {
		t.Errorf("Loaded configs don't match\nExpected\n%+vGot\n%+v\n",
			expectedConfigFiles, config.Files)
	}
}

func TestLoadMtestConfigsFileOrder(t *testing.T) {
	config1, err := LoadMtestConfigDir("../mockit/mtest")
	if err != nil {
		t.Fatalf("Failed to load config: %s", err)
	}

	config2, err := LoadMtestConfig("../mockit/partial_mtest_config")
	if err != nil {
		t.Fatalf("Failed to load config: %s", err)
	}

	expected := []string{
		// filepath.FromSlash changes these to backslash \ on Windows
		filepath.FromSlash("../mockit/mtest/common.hcl"),
		filepath.FromSlash("../mockit/mtest/service.json"),
		filepath.FromSlash("../mockit/partial_mtest_config"),
	}

	config := config1.Merge(config2)

	if !reflect.DeepEqual(config.Files, expected) {
		t.Errorf("Loaded configs don't match\nwant: %+v\n got: %+v\n",
			expected, config.Files)
	}
}

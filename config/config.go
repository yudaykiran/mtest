package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"sort"
	"strings"
)

// MtestConfig is the configuration for Mtest.
type MtestConfig struct {

	// LogLevel is the level of the logs to putout
	LogLevel string `mapstructure:"log_level"`

	// EnableSyslog is used to enable sending logs to syslog
	EnableSyslog bool `mapstructure:"enable_syslog"`

	// SyslogFacility is used to control the syslog facility used.
	SyslogFacility string `mapstructure:"syslog_facility"`

	// Version information is set at compilation time
	Revision          string
	Version           string
	VersionPrerelease string

	// List of config files that have been loaded (in order)
	Files []string `mapstructure:"-"`
}

// DefaultMtestConfig is a the baseline configuration for Mtest
func DefaultMtestConfig() *MtestConfig {
	return &MtestConfig{
		LogLevel:       "INFO",
		SyslogFacility: "LOCAL0",
	}
}

// Merge merges two configurations & returns a new one.
func (mc *MtestConfig) Merge(b *MtestConfig) *MtestConfig {
	result := *mc

	if b.LogLevel != "" {
		result.LogLevel = b.LogLevel
	}
	if b.EnableSyslog {
		result.EnableSyslog = true
	}
	if b.SyslogFacility != "" {
		result.SyslogFacility = b.SyslogFacility
	}

	// Merge config files lists
	result.Files = append(result.Files, b.Files...)

	return &result
}

// LoadMtestConfig loads the configuration at the given path, regardless if
// its a file or directory.
func LoadMtestConfig(path string) (*MtestConfig, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		return LoadMtestConfigDir(path)
	}

	cleaned := filepath.Clean(path)
	mconfig, err := ParseMtestConfigFile(cleaned)
	if err != nil {
		return nil, fmt.Errorf("Error loading %s: %s", cleaned, err)
	}

	mconfig.Files = append(mconfig.Files, cleaned)
	return mconfig, nil
}

// LoadMtestConfigDir loads all the configurations in the given directory
// in alphabetical order.
func LoadMtestConfigDir(dir string) (*MtestConfig, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf(
			"configuration path must be a directory: %s", dir)
	}

	var files []string
	err = nil
	for err != io.EOF {
		var fis []os.FileInfo
		fis, err = f.Readdir(128)
		if err != nil && err != io.EOF {
			return nil, err
		}

		for _, fi := range fis {
			// Ignore directories
			if fi.IsDir() {
				continue
			}

			// Only care about files that are valid to load.
			name := fi.Name()
			skip := true
			if strings.HasSuffix(name, ".hcl") {
				skip = false
			} else if strings.HasSuffix(name, ".json") {
				skip = false
			}
			if skip || isTemporaryFile(name) {
				continue
			}

			path := filepath.Join(dir, name)
			files = append(files, path)
		}
	}

	// Fast-path if we have no files
	if len(files) == 0 {
		return &MtestConfig{}, nil
	}

	sort.Strings(files)

	var result *MtestConfig
	for _, f := range files {
		mconfig, err := ParseMtestConfigFile(f)
		if err != nil {
			return nil, fmt.Errorf("Error loading %s: %s", f, err)
		}
		mconfig.Files = append(mconfig.Files, f)

		if result == nil {
			result = mconfig
		} else {
			result = result.Merge(mconfig)
		}
	}

	return result, nil
}

// isTemporaryFile returns true or false depending on whether the
// provided file name is a temporary file for the following editors:
// emacs or vim.
func isTemporaryFile(name string) bool {
	return strings.HasSuffix(name, "~") || // vim
		strings.HasPrefix(name, ".#") || // emacs
		(strings.HasPrefix(name, "#") && strings.HasSuffix(name, "#")) // emacs
}

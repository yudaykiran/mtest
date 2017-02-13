package logging

import (
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/go-syslog"
	"github.com/hashicorp/logutils"
	"github.com/openebs/mtest/logging/gated-writer"
)

// WriteMaker is a blueprint to build variety of log writers
type WriterVariantsMaker interface {
	Make(logLevel string, syslog bool) error
	GatedWriter() *gatedwriter.Writer
	PipedWriter() *LogWriter
	MultiWriter() io.Writer
}

type WriterVariantsMake struct {
	existing    io.Writer
	gatedWriter *gatedwriter.Writer
	pipedWriter *LogWriter
	multiWriter io.Writer
}

func (w *WriterVariantsMake) GatedWriter() *gatedwriter.Writer {
	return w.gatedWriter
}

func (w *WriterVariantsMake) PipedWriter() *LogWriter {
	return w.pipedWriter
}

func (w *WriterVariantsMake) MultiWriter() io.Writer {
	return w.multiWriter
}

func (w *WriterVariantsMake) Make(logLevel string, doSyslog bool) error {
	// Setup filtered-gated-logger !!
	// First create the gated log writer, which will buffer logs until we're ready.
	gatedLogger := &gatedwriter.Writer{
		// Set gated logger's writer against that of any existing writer
		Writer: w.existing,
	}

	// Now create the level filter, filtering logs of the specified level.
	// This filtering will be set on top of just created gated logger
	logFilter := LevelFilter()
	logFilter.MinLevel = logutils.LogLevel(strings.ToUpper(logLevel))
	logFilter.Writer = gatedLogger

	if !ValidateLevelFilter(logFilter.MinLevel, logFilter) {
		return fmt.Errorf("Invalid log level: %s. Valid log levels are: %v", logFilter.MinLevel, logFilter.Levels)
	}

	// Check if syslog is enabled
	var syslog io.Writer
	if doSyslog {
		l, err := gsyslog.NewLogger(gsyslog.LOG_NOTICE, "LOCAL0", "mtest")
		if err != nil {
			return fmt.Errorf("Syslog setup failed: %v", err)
		}

		// Setup syslog-filtered-gated-logger !!
		syslog = &SyslogWrapper{l, logFilter}
	}

	// Create a log pipe too
	// This acts as a log sink with pipes (i.e. otherwise known as handles)
	// that can be directed to various sink handlers.
	//    NOTE:
	//      These handlers can be registered (& de-registered) at a later
	//      point of time dynamically.
	logPipe := NewLogWriter(512)

	// Create a multi-write logger around all the above log variants
	var multiLogger io.Writer
	if syslog != nil {
		multiLogger = io.MultiWriter(logFilter, logPipe, syslog)
	} else {
		multiLogger = io.MultiWriter(logFilter, logPipe)
	}

	// Set all the logger variants
	w.gatedWriter = gatedLogger
	w.pipedWriter = logPipe
	w.multiWriter = multiLogger

	return nil
}

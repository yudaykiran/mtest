// Package mtest provides the entry point to trigger testing of various
// OpenEBS projects.
//
// A running OpenEBS process is known as a *Runner*.
//
// Mtest provides the structure to manage execution of a Runner
//
// Mtest makes use of Start() method to trigger a particular Runner.
//
// Interface based design has been used extensively to
// provide flexibility in design & ensure each block of code can
// be unit tested effectively.
package mtest

import (
	"fmt"
	"log"
	"sync"
)

// A Report is the Runner's run report structure for individual report fields.
type Report struct {

	// The runner that ran the use-case
	Runner string

	// Use case name
	Usecase string

	// Run's entire response can be embedded here
	Message interface{}

	// A descriptive status used when Message is nil
	Status string

	// Overall run status as a flag
	Success bool
}

// `Runner` is the interface for any OpenEBS program. A Runner will
// use `Mtest drivers` to test an OpenEBS program.
type Runner interface {

	// Name returns the name of the Runner
	Name() string

	// Run returns a run report on successful execution of test case(s).
	// Error is returned if the report were not obtainable, due to issues.
	Run() ([]*Report, error)

	// IsParallel indicates if test cases within a Run() can be
	// executed in parallel. The assumption is there are multiple test cases
	// to be executed against an OpenEBS program.
	IsParallel() bool

	// IsComplete indicates if the runner has completed execution of its
	// test cases against the OpenEBS program.
	IsComplete() bool

	// The logger that Runner is using currently.
	// This one will be used by Mtest too.
	Logger() *log.Logger

	// Stops the runner
	Stop()
}

// `Parallel` provides shared parallelism logic to be used by Mtest
// runners to execute test cases in a parallel manner.
//
// Usage:
//    this struct as an anonymous field within the runner's struct.
//
// Example:
//     type JivaRunner struct {
//         Parallel
//         ...
//     }
type Parallel struct {
	// TODO No of goroutines ??
	forks int
}

// SetParallelism sets the parallel IsParallel will check when called.
//
// If forks is greater than 0 then the runner will execute test cases
// in parallel.
func (p *Parallel) SetParallelism(forks int) {
	p.forks = forks
}

// IsParallel returns if the no of forks is greater than 0
func (p *Parallel) IsParallel() bool {
	if p.forks > 0 {
		return true
	}
	return false
}

// Blueprint to create Mtest structure.
type MtestMaker interface {
	Make() (*Mtest, error)
}

// A structure that abides by MtestMaker blueprint
type MtestMake struct {
	runner Runner
	logger *log.Logger
}

// The interface method
func (t *MtestMake) Make() (*Mtest, error) {
	if t.runner == nil {
		return nil, fmt.Errorf("Runner is required to create Mtest")
	}

	if t.runner.Logger() != nil {
		t.logger = t.runner.Logger()
	}

	if t.logger == nil {
		return nil, fmt.Errorf("Logger required to create Mtest")
	}

	return &Mtest{
		logger:  t.logger,
		runner:  t.runner,
		running: false,
	}, nil
}

// Mtest provides a synchronous start of runners.
//
// Mtest is safe to use across multiple goroutines and will manage to
// provide the current status of execution via report.
type Mtest struct {
	logger  *log.Logger
	reports []*Report
	running bool
	m       sync.Mutex

	runner Runner
}

// A setter method that can change the runner at run-time.
// There are some checks which if satisfied leads to overriding
// of old runner.
//
// Usage:
//    There are multiple usages starting from
//    ability to unit test Mtest methods to managing future
//    requirements.
func (t *Mtest) SetRunner(newRunner Runner) (Runner, error) {
	t.m.Lock()
	defer t.m.Unlock()

	old := t.runner

	if !t.isRunning() {
		t.runner = newRunner
		return old, nil
	}

	return nil, fmt.Errorf("Can not set a new runner when mtest is running")
}

// Start will start this Mtest's associated runner, will return
// the result as reports, or error if the runner failed.
func (t *Mtest) Start() ([]*Report, error) {
	t.m.Lock()
	defer t.m.Unlock()

	// If not running then run
	if !t.isRunning() {
		t.running = true
		reports, err := t.runner.Run()
		if err != nil {
			return nil, err
		}

		t.reports = reports
	} else {
		return nil, fmt.Errorf("Mtest is already running")
	}

	return t.reports, nil
}

// Signals a stop of mtest
func (t *Mtest) Stop() {
	t.m.Lock()
	defer t.m.Unlock()

	t.runner.Stop()
	t.running = false
}

// IsRunning is a public safe version of isRunning()
func (t *Mtest) IsRunning() bool {
	t.m.Lock()
	defer t.m.Unlock()

	return t.isRunning()
}

// isRunning helper method wrapping the definition of completion of
// testsuite and runner.
func (t *Mtest) isRunning() bool {
	return t.running || !t.runner.IsComplete()
}

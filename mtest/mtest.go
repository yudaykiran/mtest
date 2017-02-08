// Package mtest provides the entry point to trigger testing of various
// OpenEBS projects.
//
// OpenEBS projects are known as **Runners**.
//
// Mtest provides the structure to manage execution of a Runner
//
// Mtest makes use of Start() method to trigger a particular Runner.
//
// Example of creating a particular Mtest:
//
//     mt := NewMtest(&GotgtRunner{})
//     report, err := mt.Start()
//
// Creating a new Runner
//
// To build a custom Runner just create a type which satisfies the Runner
// interface and pass it to the NewMtest method.
//
//     type GotgtRunner struct{}
//     func (r *GotgtRunner) Run() (*Report, error) {...}
//
//     suite := NewMtest(&GotgtRunner{})
//     report, err := suite.Start()
//     if err == nil {
//        return err
//     }
//
package mtest

import (
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

// Mtest provides a synchronous start of runners.
//
// Mtest is safe to use across multiple goroutines and will manage to
// provide the current status of execution via report.
type Mtest struct {
	reports []*Report
	running bool
	m       sync.Mutex

	runner Runner
}

// NewMtest returns a pointer to a new Mtest with the runner.
func NewMtest(runner Runner) *Mtest {
	return &Mtest{
		runner:  runner,
		running: false,
	}
}

// Start returns the report value, or error if the runner failed to run
func (t *Mtest) Start() ([]*Report, error) {
	t.m.Lock()
	defer t.m.Unlock()

	if t.isRunning() {
		reports, err := t.runner.Run()
		if err != nil {
			return nil, err
		}

		t.reports = reports
		t.running = true
	}

	return t.reports, nil
}

// Stop stops the mtest.
//
// TODO
// This can be used to signal the runners to stop s.t each runner can
// arrive at a safe state before actually stopping.
func (t *Mtest) Stop() {
	t.m.Lock()
	defer t.m.Unlock()

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
	return t.running || t.runner.IsComplete()
}

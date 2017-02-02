// The testsuite package has some of its design elements adapted from following
// libraries:
//
//    - aws/aws-sdk-go
//
// NOTE :
//    Design & structure will change as the current code is in pre alpha stage
//
// Package testsuite manages execution of functional tests of various
// openebs projects, henceforth known as Runners.
//
// The Testsuite is the primary method of managing execution of functional
// tests.
//
// Testsuite will make use of Start() method to execute functional tests against
// a particular Runner.
//
// Example of creating a particular testsuite
//
//     suite := NewMserverTestSuite()
//
//     // Retrieve the credentials value
//     report, err := suite.Exec()
//     if err != nil {
//         // handle error
//     }
//
// Example of an alternative coding style:
//
//     suite := NewTestsuite(&GotgtRunner{})
//     runReport, err := suite.Start()
//
// Creating a new Runner
//
// To build a custom Runner just create a type which satisfies the Runner
// interface and pass it to the NewTestsuite method.
//
//     type GotgtRunner struct{}
//     func (r *GotgtRunner) Run() (CredValue, error) {...}
//
//     suite := NewTestsuite(&GotgtRunner{})
//     report, err := suite.Start()
//     if err == nil {
//        return err
//     }
//
package testsuite

import (
	"sync"
)

// A Report is the Runner's run report struct for individual report fields.
type Report struct {

	// Test case name
	Name string

	// Run message
	Message string

	// Run status
	Status string

	// Run flag indicating success or failed
	Success bool
}

// A Runner is the interface for any component which will run functional
// tests of that component.
type Runner interface {
	// Run returns nil error if it successfully retrieved the report.
	// Error is returned if the report were not obtainable, or empty.
	Run() (*Report, error)

	// IsParallel indicates if multiple test cases within a Run() can be
	// executed in parallel
	IsParallel() bool

	// IsComplete indicates if the runner has completed execution of its
	// test cases
	IsComplete() bool
}

// A Parallel provides shared parallelism logic to be used by testsuite
// runners to implement parallel functionality.
//
// Usage:
//    this struct as an anonymous field within the runner's struct.
//    this struct as an anonymous field within the testsuite's struct.
//
// Example:
//     type JivaRunner struct {
//         Parallel
//         ...
//     }
type Parallel struct {
	// No of goroutines
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

// A Testsuite provides a synchronous start of runners.
//
// Testsuite is safe to use across multiple goroutines and will manage to
// provide the current status of execution via report.
type Testsuite struct {
	report  *Report
	running bool
	m       sync.Mutex

	runner Runner
	Parallel
}

// NewTestsuite returns a pointer to a new Testsuite with the runner.
func NewTestsuite(runner Runner) *Testsuite {
	return &Testsuite{
		runner:  runner,
		running: false,
	}
}

// Start returns the report value, or error if the runner failed to run
//
// TODO
// There will be structural changes that takes into account `start
// of multiple runners in parallel` & `each runner running multiple
// test cases in parallel`.
//
// TODO
// Might adapt aws-sdk-go credentials' chain concept & config
func (t *Testsuite) Start() (*Report, error) {
	t.m.Lock()
	defer t.m.Unlock()

	if t.isRunning() {
		report, err := t.runner.Run()
		if err != nil {
			return nil, err
		}
		t.report = report
		t.running = true
	}

	return t.report, nil
}

// Stop stops the testsuite.

// TODO
// This can be used to signal the runners to stop s.t each runner can
// arrive at a safe state before actually stopping.
func (t *Testsuite) Stop() {
	t.m.Lock()
	defer t.m.Unlock()

	t.running = false
}

// IsRunning is a public safe version of isRunning()
func (t *Testsuite) IsRunning() bool {
	t.m.Lock()
	defer t.m.Unlock()

	return t.isRunning()
}

// isRunning helper method wrapping the definition of completion of testsuite
// and runner.
func (t *Testsuite) isRunning() bool {
	return t.running || t.runner.IsComplete()
}

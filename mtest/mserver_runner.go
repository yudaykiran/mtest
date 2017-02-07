package mtest

import (
	"os"
	"strconv"
)

// MserverRunnerName provides a name of Mserver runner
const MserverRunnerName = "MserverRunner"

// A MserverTest defines the structure of a Mserver test case
type MserverTest struct {
}

// A MserverRunner structure definition
type MserverRunner struct {
	Parallel
	inprogress bool
}

// NewMserverMtest returns a pointer to a new Mtest object
// wrapping the Mserver runner. A convenience function.
func NewMserverMtest() *Mtest {
	return NewMtest(&MserverRunner{})
}

// Run runs the test cases agains a running Mserver process
//
// TODO
// This code will change when actual parallelism is implemented
// Concept of mutexes & goroutines may become prominent
//
// Alternatively, we may use some go testing library that does
// fork & join using goroutines.
func (r *MserverRunner) Run() (*Report, error) {
	r.inprogress = true

	runThreads := os.Getenv("MSERVER_RUNNER_THREADS")
	if runThreads == "" {
		runThreads = "0"
	}

	// basic base-10 string parse
	threads, err := strconv.Atoi(runThreads)
	if err != nil {
		return nil, err
	}

	r.Parallel = Parallel{
		forks: threads,
	}

	// TODO
	// Do the real run of test cases
	r.runTestCases()

	r.inprogress = false

	return &Report{
		Name: MserverRunnerName,
	}, nil
}

// IsComplete returns if the runner has executed all its
// test cases.
func (r *MserverRunner) IsComplete() bool {
	return !r.inprogress
}

func (r *MserverRunner) runTestCases() (*Report, error) {
	return nil, nil
}

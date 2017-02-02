package testsuite

import (
	"os"
	"strconv"
)

// MserverRunnerName provides a name of Mserver runner
const MserverRunnerName = "MserverRunner"

// A MserverRunnerName structure definition
type MserverRunner struct {
	parallel   bool
	inprogress bool
}

// NewMserverTestsuite returns a pointer to a new Testsuite object
// wrapping the Mserver runner. A convenience function.
func NewMserverTestsuite() *Testsuite {
	return NewTestsuite(&MserverRunner{})
}

// Run runs the test cases agains a running Mserver process
//
// TODO
// This code will change when actual parallelism is implemented
// Concept of mutexes & goroutines will be prominent
//
// Alternatively, we may use some go testing library that does
// fork & join using goroutines.
func (r *MserverRunner) Run() (*Report, error) {
	r.inprogress = true

	envPl := os.Getenv("MSERVER_RUNNER_PARALLEL")
	if envPl == "" {
		envPl = "false"
	}

	parallel, err := strconv.ParseBool(envPl)
	if err != nil {
		return nil, err
	}

	r.parallel = parallel

	// TODO
	// Do the real run of test cases

	return &Report{
		Name: MserverRunnerName,
	}, nil
}

// IsParallel returns if the runner will execute test cases
// in a parallel manner.
func (r *MserverRunner) IsParallel() bool {
	return r.parallel
}

// IsComplete returns if the runner has executed all its
// test cases.
func (r *MserverRunner) IsComplete() bool {
	return !r.inprogress
}

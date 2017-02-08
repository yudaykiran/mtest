package mtest

import (
	"os"
	"strconv"

	"github.com/openebs/mtest/driver"
	"github.com/openebs/mtest/driver/ebs"
)

// MserverRunnerName provides a name of Mserver runner
const (
	// Name of this mtest runner
	MTEST_MSERVER_RUNNER_NAME = "mserver.runner"

	// Constant to name the volume removal use-case
	MSERVER_VOLUME_REMOVE_USECASE = "mserver.volume.remove.usecase"
)

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

func (r *MserverRunner) Name() string {
	return MTEST_MSERVER_RUNNER_NAME
}

// Run runs the use cases against a running Mserver process
func (r *MserverRunner) Run() ([]*Report, error) {
	r.start()
	defer r.stop()

	runThreads := os.Getenv("MTEST_MSERVER_RUNNER_THREADS")
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

	// Do the real run of use cases
	return r.runUseCases()
}

// IsComplete returns if the runner has executed all its
// test cases.
func (r *MserverRunner) IsComplete() bool {
	return !r.inprogress
}

func (r *MserverRunner) start() {
	r.inprogress = true
}

func (r *MserverRunner) stop() {
	r.inprogress = false
}

func (r *MserverRunner) runUseCases() ([]*Report, error) {

	ebsDriver, err := driver.GetDriver(ebs.DRIVER_NAME, "", make(map[string]string))
	if err != nil {
		return nil, err
	}

	usecases := []string{ebs.EBS_VOLUME_REMOVE_EXEC}

	mapExecs, err := ebsDriver.Executors(usecases...)
	if err != nil {
		return nil, err
	}

	volRemover := mapExecs[ebs.EBS_VOLUME_REMOVE_EXEC]

	resp, err := volRemover.Exec(driver.Request{
		Name: "vol1",
	})

	reports := make([]*Report, len(usecases))

	if err != nil {
		report := &Report{
			Runner:  MTEST_MSERVER_RUNNER_NAME,
			Usecase: MSERVER_VOLUME_REMOVE_USECASE,
			Message: resp,
			Status:  "OK",
			Success: true,
		}

		reports = append(reports, report)

	} else {
		report := &Report{
			Runner:  MTEST_MSERVER_RUNNER_NAME,
			Usecase: MSERVER_VOLUME_REMOVE_USECASE,
			Message: err,
			Status:  "FAILED",
			Success: false,
		}

		reports = append(reports, report)
	}

	return reports, nil
}

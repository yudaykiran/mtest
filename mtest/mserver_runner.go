package mtest

import (
	"fmt"
	"io"
	"log"
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

	// Constant to name the volume creation use-case
	MSERVER_VOLUME_CREATE_USECASE = "mserver.volume.create.usecase"
)

// A MserverRunner structure definition
type MserverRunner struct {
	logger *log.Logger
	Parallel
	inprogress bool
}

// NewMserverRunMaker returns an instance of MtestMake that
// aligns to MtestMaker interface.
func NewMserverRunMaker(logWriter io.Writer) (MtestMaker, error) {

	if logWriter == nil {
		return nil, fmt.Errorf("Log writer is required to create a MServerRunner")
	}

	return &MtestMake{
		runner: &MserverRunner{
			logger:     log.New(logWriter, "", log.LstdFlags|log.Lmicroseconds),
			inprogress: false,
		},
	}, nil
}

func (r *MserverRunner) Name() string {
	return MTEST_MSERVER_RUNNER_NAME
}

func (r *MserverRunner) Logger() *log.Logger {
	return r.logger
}

// Run runs the use cases against a running Mserver process
func (r *MserverRunner) Run() ([]*Report, error) {
	r.Start()
	defer r.Stop()

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

func (r *MserverRunner) Start() {
	r.inprogress = true
}

func (r *MserverRunner) Stop() {
	r.inprogress = false
}

func (r *MserverRunner) runUseCases() ([]*Report, error) {

	// Get the ebs driver
	ebsDriver, err := driver.GetDriver(ebs.DRIVER_NAME, "", make(map[string]string))
	if err != nil {
		return nil, err
	}

	// The usecases can optionally be sent by the caller/client
	// TODO
	//    The usecase names & corresponding executions should be invoked
	// in sequence with various options/flags to handle this **chain of
	// execution**.
	usecases := []string{ebs.EBS_VOLUME_CREATE_EXEC}

	// Get the executors corresponding to each use-case
	mapExecs, err := ebsDriver.Executors(usecases...)
	if err != nil {
		return nil, err
	}

	// The volume executor
	volCtr := mapExecs[ebs.EBS_VOLUME_CREATE_EXEC]

	// Execute
	resp, err := volCtr.Exec(driver.Request{
		Name: "vol1",
		//Options: map[string]string{
		//},
	})

	reports := make([]*Report, len(usecases))

	if err == nil {
		report := &Report{
			Runner:  MTEST_MSERVER_RUNNER_NAME,
			Usecase: MSERVER_VOLUME_CREATE_USECASE,
			Message: resp,
			Status:  "OK",
			Success: true,
		}

		reports = append(reports, report)

	} else {
		report := &Report{
			Runner:  MTEST_MSERVER_RUNNER_NAME,
			Usecase: MSERVER_VOLUME_CREATE_USECASE,
			Message: err.Error(),
			Status:  "FAILED",
			Success: false,
		}

		reports = append(reports, report)
	}

	return reports, nil
}

// Package driver has part of its design influences from
// rancher/convoy project.
//
// `driver` package defines:
//    1. a set of contracts &
//    2. a set of structures.
//
// A contract is technically known as an interface while
// a structure is called as a struct.
//
// A concrete Mtest Driver can be registered & acted upon.
// In addition, this driver implementation can be directed
// to provide one or more executors based on hints.
//
// Hints are simply a set of variadic strings meant to be
// passed by the callers to driver's functions.
package driver

import (
	"fmt"
	"path/filepath"

	"github.com/Sirupsen/logrus"
)

// InitFunc is the initialize function for each Mtest Driver.
// Each driver must implement this function and register itself
// through Register().
type InitFunc func(root string, config map[string]string) (MtestDriver, error)

// Executor interface is a contract that defines the execution.
// In simpler terms, it is meant to define the exact use-case.
type Executor interface {
	Exec(req Request) (*Response, error)
}

// MtestDriver interface is a contract that needs to be
// implemented by various Mtest Driver implementors.
//
// A concrete Mtest Driver can directed to provide
// specific executors.
//
// e.g. EBS Mtest Driver can be directed to provide below executors:
//  1. CreateVolume
//  2. DeleteVolume
type MtestDriver interface {
	// Provide a unique name of the MtestDriver implementor
	Name() string

	// Provide additional info about the Mtest Driver implementor
	Info() (map[string]string, error)

	// Fetches executors based on provided hints.
	Executors(hints ...string) (map[string]Executor, error)
}

// Request is used for passing data required during execution of a
// particular use-case
type Request struct {
	Name    string
	Options map[string]string
}

// Response is used to give back the response after the execution of
// the use-case
type Response struct {
	Values map[string]interface{}
}

var (
	initializers map[string]InitFunc
	log          = logrus.WithFields(logrus.Fields{"pkg": "mtest.driver"})
)

func init() {
	initializers = make(map[string]InitFunc)
}

// Register registers drivers, i.e. add implemented drivers via `InitFunc`
// to the known driver list. `InitFunc` is supposed to be defined in
// individual Mtest Driver implementation.
func Register(name string, initFunc InitFunc) error {
	_, exists := initializers[name]

	if exists {
		return fmt.Errorf("MtestDriver %s has already been registered", name)
	}

	initializers[name] = initFunc
	return nil
}

// GetDriver would be called each time when a Mtest Driver instance is needed.
func GetDriver(name, root string, config map[string]string) (MtestDriver, error) {
	_, exists := initializers[name]

	if !exists {
		return nil, fmt.Errorf("MtestDriver '%v' is not supported!", name)
	}

	drvRoot := filepath.Join(root, name)

	// The actual invocation of driver initialization
	return initializers[name](drvRoot, config)
}

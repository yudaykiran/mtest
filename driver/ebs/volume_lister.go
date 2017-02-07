package ebs

import (
	"fmt"

	"github.com/openebs/mtest/driver"
)

const (
	// Name of this executor
	// This executor will be known as this to the outside world
	EBS_VOLUME_LIST_EXEC = "ebs.volume.list.executor"
)

// This is a EBS driver executor.
type VolumeLister struct {
	d *EBSDriver
}

func init() {
	// Register by passing the name of this executor
	// and its initializing function definition.
	RegisterAsEBSExecutor(EBS_VOLUME_LIST_EXEC, VolumeListerInit)
}

// The initializing function of VolumeLister executor.
func VolumeListerInit(ebsDriver *EBSDriver) (driver.Executor, error) {
	return &VolumeLister{
		d: ebsDriver,
	}, nil
}

func (v *VolumeLister) Exec(req driver.Request) (*driver.Response, error) {

	volumeIDs, err := v.d.listVolumeNames()
	if err != nil {
		return nil, err
	}

	execs, err := v.d.Executors(EBS_VOLUME_READ_EXEC)
	if err != nil {
		return nil, err
	}

	_, exists := execs[EBS_VOLUME_READ_EXEC]

	if !exists {
		return nil, fmt.Errorf("Volume reader %v not found", EBS_VOLUME_READ_EXEC)
	}

	req2 := driver.Request{
		Name: req.Name,
	}

	values := make(map[string]interface{})

	for _, uuid := range volumeIDs {

		req2.Options = map[string]string{
			"uuid": uuid,
		}

		resp, err := execs[EBS_VOLUME_READ_EXEC].Exec(req2)
		if err != nil {
			return nil, err
		}

		values[uuid] = resp.Values
	}

	return &driver.Response{
		Values: values,
	}, nil
}

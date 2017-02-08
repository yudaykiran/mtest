package ebs

import (
	"github.com/openebs/mtest/driver"
	"github.com/openebs/mtest/util"
)

const (
	// Name of this executor
	// This executor will be known as this to the outside world
	EBS_SNAPSHOT_READ_EXEC = "ebs.snapshot.read.executor"
)

// This is a EBS driver executor.
type SnapshotReader struct {
	d *EBSDriver
}

func init() {
	// Register by passing the name of this executor
	// and its initializing function definition.
	RegisterAsEBSExecutor(EBS_SNAPSHOT_READ_EXEC, SnapshotReaderInit)
}

// The initializing function of VolumeReader executor.
func SnapshotReaderInit(ebsDriver *EBSDriver) (driver.Executor, error) {
	return &SnapshotReader{
		d: ebsDriver,
	}, nil
}

func (s *SnapshotReader) Exec(req driver.Request) (*driver.Response, error) {

	s.d.mutex.Lock()
	defer s.d.mutex.Unlock()

	id := req.Name
	volumeID, err := util.GetFieldFromOpts(OPT_VOLUME_NAME, req.Options)
	if err != nil {
		return nil, err
	}

	info, err := s.d.getSnapshotInfo(id, volumeID)
	if err != nil {
		return nil, err
	}

	values := map[string]interface{}{
		id: info,
	}

	return &driver.Response{
		Values: values,
	}, nil
}

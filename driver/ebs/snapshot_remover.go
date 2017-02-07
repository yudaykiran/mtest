package ebs

import (
	"github.com/openebs/mtest/driver"
	"github.com/openebs/mtest/util"
)

const (
	// Name of this executor
	// This executor will be known as this to the outside world
	EBS_SNAP_REMOVE_EXEC = "ebs.snapshot.remove.executor"
)

// This is a EBS driver executor.
type SnapshotRemover struct {
	d *EBSDriver
}

func init() {
	// Register by passing the name of this executor
	// and its initializing function definition.
	RegisterAsEBSExecutor(EBS_SNAP_REMOVE_EXEC, SnapshotRemoverInit)
}

// The initializing function of SnapshotCreator executor.
func SnapshotRemoverInit(ebsDriver *EBSDriver) (driver.Executor, error) {
	return &SnapshotRemover{
		d: ebsDriver,
	}, nil
}

func (s *SnapshotRemover) Exec(req driver.Request) (*driver.Response, error) {

	s.d.mutex.Lock()
	defer s.d.mutex.Unlock()

	id := req.Name
	volumeID, err := util.GetFieldFromOpts(OPT_VOLUME_NAME, req.Options)
	if err != nil {
		return nil, err
	}

	snapshot, volume, err := s.d.getSnapshotAndVolume(id, volumeID)
	if err != nil {
		return nil, err
	}

	log.Debugf("Removing snapshot %v(%v) of volume %v(%v)", id, snapshot.EBSID, volumeID, volume.EBSID)

	delete(volume.Snapshots, id)

	err = util.ObjectSave(volume)
	if err != nil {
		return nil, err
	}

	return &driver.Response{}, nil
}

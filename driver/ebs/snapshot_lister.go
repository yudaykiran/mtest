package ebs

import (
	"github.com/openebs/mtest/driver"
	"github.com/openebs/mtest/util"
)

const (
	// Name of this executor
	// This executor will be known as this to the outside world
	EBS_SNAPSHOT_LIST_EXEC = "ebs.snapshot.list.executor"
)

// This is a EBS driver executor.
type SnapshotLister struct {
	d *EBSDriver
}

func init() {
	// Register by passing the name of this executor
	// and its initializing function definition.
	RegisterAsEBSExecutor(EBS_SNAPSHOT_LIST_EXEC, SnapshotListerInit)
}

// The initializing function of this executor.
func SnapshotListerInit(ebsDriver *EBSDriver) (driver.Executor, error) {
	return &SnapshotLister{
		d: ebsDriver,
	}, nil
}

func (s *SnapshotLister) Exec(req driver.Request) (*driver.Response, error) {

	s.d.mutex.Lock()
	defer s.d.mutex.Unlock()

	var (
		volumeIDs []string
		err       error
	)

	specifiedVolumeID, _ := util.GetFieldFromOpts(OPT_VOLUME_NAME, req.Options)

	if specifiedVolumeID != "" {
		volumeIDs = []string{
			specifiedVolumeID,
		}
	} else {
		volumeIDs, err = s.d.listVolumeNames()
		if err != nil {
			return nil, err
		}
	}

	values := make(map[string]interface{})

	for _, volumeID := range volumeIDs {
		volume := s.d.blankVolume(volumeID)

		err := util.ObjectLoad(volume)
		if err != nil {
			return nil, err
		}

		for snapshotID := range volume.Snapshots {
			values[snapshotID], err = s.d.getSnapshotInfo(snapshotID, volumeID)
			if err != nil {
				return nil, err
			}
		}
	}

	return &driver.Response{
		Values: values,
	}, nil
}

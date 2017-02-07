package ebs

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/openebs/mtest/driver"
	. "github.com/openebs/mtest/logging"
	"github.com/openebs/mtest/util"
)

const (
	// Name of this executor
	// This executor will be known as this to the outside world
	EBS_SNAP_CREATE_EXEC = "ebs.snapshot.create.executor"
)

// This is a EBS driver executor.
type SnapshotCreator struct {
	d *EBSDriver
}

func init() {
	// Register by passing the name of this executor
	// and its initializing function definition.
	RegisterAsEBSExecutor(EBS_SNAP_CREATE_EXEC, SnapshotCreatorInit)
}

// The initializing function of SnapshotCreator executor.
func SnapshotCreatorInit(ebsDriver *EBSDriver) (driver.Executor, error) {
	return &SnapshotCreator{
		d: ebsDriver,
	}, nil
}

func (s *SnapshotCreator) Exec(req driver.Request) (*driver.Response, error) {

	s.d.mutex.Lock()
	defer s.d.mutex.Unlock()

	id := req.Name
	volumeID, err := util.GetFieldFromOpts(OPT_VOLUME_NAME, req.Options)
	if err != nil {
		return nil, err
	}

	volume := s.d.blankVolume(volumeID)
	if err := util.ObjectLoad(volume); err != nil {
		return nil, err
	}

	snapshot, exists := volume.Snapshots[id]
	if exists {
		return nil, generateError(logrus.Fields{
			LOG_FIELD_VOLUME:   volumeID,
			LOG_FIELD_SNAPSHOT: id,
		}, "Snapshot already exists with uuid")
	}

	tags := map[string]string{
		"MtestVolumeName":   volumeID,
		"MtestSnapshotName": id,
	}

	request := &CreateSnapshotRequest{
		VolumeID:    volume.EBSID,
		Description: fmt.Sprintf("Mtest volume snapshot"),
		Tags:        tags,
	}
	ebsSnapshotID, err := s.d.client.CreateSnapshot(request)

	if err != nil {
		return nil, err
	}
	log.Debugf("Created snapshot %v(%v) of volume %v(%v)", id, ebsSnapshotID, volumeID, volume.EBSID)

	snapshot = Snapshot{
		Name:       id,
		VolumeName: volumeID,
		EBSID:      ebsSnapshotID,
	}
	volume.Snapshots[id] = snapshot

	err = util.ObjectSave(volume)
	if err != nil {
		return nil, err
	}

	values := make(map[string]interface{})
	values["snap"] = snapshot

	return &driver.Response{
		Values: values,
	}, nil
}

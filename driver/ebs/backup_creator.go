package ebs

import (
	"fmt"

	"github.com/openebs/mtest/driver"
)

const (
	// Name of this executor
	// This executor will be known as this to the outside world
	EBS_BACKUP_CREATE_EXEC = "ebs.backup.create.executor"
)

// This is a EBS driver executor.
type BackupCreator struct {
	d *EBSDriver
}

func init() {
	// Register by passing the name of this executor
	// and its initializing function definition.
	RegisterAsEBSExecutor(EBS_BACKUP_CREATE_EXEC, BackupCreatorInit)
}

// The initializing function of this executor.
func BackupCreatorInit(ebsDriver *EBSDriver) (driver.Executor, error) {
	return &BackupCreator{
		d: ebsDriver,
	}, nil
}

func (b *BackupCreator) Exec(req driver.Request) (*driver.Response, error) {
	_, snapExists := req.Options[OPT_SNAPSHOT_ID]
	if !snapExists {
		return nil, fmt.Errorf("Snapshot ID not provided")
	}

	_, volExists := req.Options[OPT_VOLUME_ID]
	if !volExists {
		return nil, fmt.Errorf("Volume ID not provided")
	}

	snapshotID := req.Options[OPT_SNAPSHOT_ID]
	volumeID := req.Options[OPT_VOLUME_ID]

	snapshot, _, err := b.d.getSnapshotAndVolume(snapshotID, volumeID)
	if err != nil {
		return nil, err
	}

	if err := b.d.client.WaitForSnapshotComplete(snapshot.EBSID); err != nil {
		return nil, err
	}

	values := make(map[string]interface{})
	values[OPT_BACKUP_URL] = encodeURL(b.d.client.Region, snapshot.EBSID)

	return &driver.Response{
		Values: values,
	}, nil
}

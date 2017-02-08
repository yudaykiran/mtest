package ebs

import (
	"fmt"

	"github.com/openebs/mtest/driver"
)

const (
	// Name of this executor
	// This executor will be known as this to the outside world
	EBS_BACKUP_REMOVE_EXEC = "ebs.backup.remove.executor"
)

// This is a EBS driver executor.
type BackupRemover struct {
	d *EBSDriver
}

func init() {
	// Register by passing the name of this executor
	// and its initializing function definition.
	RegisterAsEBSExecutor(EBS_BACKUP_REMOVE_EXEC, BackupRemoverInit)
}

// The initializing function of this executor.
func BackupRemoverInit(ebsDriver *EBSDriver) (driver.Executor, error) {
	return &BackupRemover{
		d: ebsDriver,
	}, nil
}

func (b *BackupRemover) Exec(req driver.Request) (*driver.Response, error) {

	_, exists := req.Options[OPT_BACKUP_URL]
	if !exists {
		return nil, fmt.Errorf("Backup URL not provided")
	}

	backupURL := req.Options[OPT_BACKUP_URL]

	region, ebsSnapshotID, err := decodeURL(backupURL)
	if err != nil {
		return nil, err
	}

	err = b.d.client.DeleteSnapshotWithRegion(ebsSnapshotID, region)
	if err != nil {
		return nil, err
	}

	return &driver.Response{}, nil
}

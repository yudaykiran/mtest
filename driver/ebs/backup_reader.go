package ebs

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/openebs/mtest/driver"
)

const (
	// Name of this executor
	// This executor will be known as this to the outside world
	EBS_BACKUP_READ_EXEC = "ebs.backup.read.executor"
)

// This is a EBS driver executor.
type BackupReader struct {
	d *EBSDriver
}

func init() {
	// Register by passing the name of this executor
	// and its initializing function definition.
	RegisterAsEBSExecutor(EBS_BACKUP_READ_EXEC, BackupReaderInit)
}

// The initializing function of this executor.
func BackupReaderInit(ebsDriver *EBSDriver) (driver.Executor, error) {
	return &BackupReader{
		d: ebsDriver,
	}, nil
}

func (b *BackupReader) Exec(req driver.Request) (*driver.Response, error) {

	_, exists := req.Options[OPT_BACKUP_URL]
	if !exists {
		return nil, fmt.Errorf("Backup URL not provided")
	}

	backupURL := req.Options[OPT_BACKUP_URL]

	region, ebsSnapshotID, err := decodeURL(backupURL)
	if err != nil {
		return nil, err
	}

	ebsSnapshot, err := b.d.client.GetSnapshotWithRegion(ebsSnapshotID, region)
	if err != nil {
		return nil, err
	}

	values := map[string]interface{}{
		"Region":        region,
		"EBSSnapshotID": aws.StringValue(ebsSnapshot.SnapshotId),
		"EBSVolumeID":   aws.StringValue(ebsSnapshot.VolumeId),
		"KmsKeyId":      aws.StringValue(ebsSnapshot.KmsKeyId),
		"StartTime":     (*ebsSnapshot.StartTime).Format(time.RubyDate),
		"Size":          strconv.FormatInt(*ebsSnapshot.VolumeSize*GB, 10),
		"State":         aws.StringValue(ebsSnapshot.State),
	}

	return &driver.Response{
		Values: values,
	}, nil
}

package ebs

import (
	"fmt"

	"github.com/openebs/mtest/driver"
	"github.com/openebs/mtest/util"
)

const (
	// Name of this executor
	// This executor will be known as this to the outside world
	EBS_VOLUME_CREATE_EXEC = "ebs.volume.create.executor"
)

// This is a EBS driver executor.
type VolumeCreator struct {
	d *EBSDriver
}

func init() {
	// Register by passing the name of this executor
	// and its initializing function definition.
	RegisterAsEBSExecutor(EBS_VOLUME_CREATE_EXEC, VolumeCreatorInit)
}

// The initializing function of VolumeCreator executor.
func VolumeCreatorInit(ebsDriver *EBSDriver) (driver.Executor, error) {
	return &VolumeCreator{
		d: ebsDriver,
	}, nil
}

// This is a loaded function, that caters to various forms of
// ebs volume creation. In addition, attaching the volume to a
// device & formatting it against a filesystem.
func (v *VolumeCreator) Exec(req driver.Request) (*driver.Response, error) {
	var (
		err        error
		volumeSize int64
		format     bool
	)

	v.d.mutex.Lock()
	defer v.d.mutex.Unlock()

	id := req.Name
	opts := req.Options

	volume := v.d.blankVolume(id)
	exists, err := util.ObjectExists(volume)

	if err != nil {
		return nil, err
	}

	if exists {
		return nil, fmt.Errorf("Volume %v already exists", id)
	}

	// EBS volume ID
	volumeID := opts[OPT_VOLUME_DRIVER_ID]
	backupURL := opts[OPT_BACKUP_URL]
	if backupURL != "" && volumeID != "" {
		return nil, fmt.Errorf("Cannot specify both backup and EBS volume ID")
	}

	newTags := map[string]string{
		"Name": id,
	}

	if volumeID != "" {

		// FETCH an EXISTING EBS volume
		ebsVolume, err := v.d.client.GetVolume(volumeID)

		if err != nil {
			return nil, err
		}

		volumeSize = *ebsVolume.Size * GB
		log.Debugf("Found EBS volume %v for volume %v, update tags", volumeID, id)

		if err := v.d.client.AddTags(volumeID, newTags); err != nil {
			log.Debugf("Failed to update tags for volume %v, but continue", volumeID)
		}

	} else if backupURL != "" {

		// RESTORE an EBS volume from an EXISTING SNAPSHOT
		region, ebsSnapshotID, err := decodeURL(backupURL)

		if err != nil {
			return nil, err
		}

		if region != v.d.client.Region {
			// We don't want to automatically copy snapshot here
			// because it's way too time consuming.
			return nil, fmt.Errorf("Snapshot %v is at %v rather than current region %v. Copy snapshot is needed",
				ebsSnapshotID, region, v.d.client.Region)
		}

		if err := v.d.client.WaitForSnapshotComplete(ebsSnapshotID); err != nil {
			return nil, err
		}
		log.Debugf("Snapshot %v is ready", ebsSnapshotID)

		ebsSnapshot, err := v.d.client.GetSnapshot(ebsSnapshotID)
		if err != nil {
			return nil, err
		}

		snapshotVolumeSize := *ebsSnapshot.VolumeSize * GB
		volumeSize, err = v.d.getSize(opts, snapshotVolumeSize)
		if err != nil {
			return nil, err
		}

		if volumeSize < snapshotVolumeSize {
			return nil, fmt.Errorf("Volume size cannot be less than snapshot size %v", snapshotVolumeSize)
		}

		volumeType, iops, err := v.d.getTypeAndIOPS(opts)
		if err != nil {
			return nil, err
		}

		r := &CreateEBSVolumeRequest{
			Size:       volumeSize,
			SnapshotID: ebsSnapshotID,
			VolumeType: volumeType,
			IOPS:       iops,
			Tags:       newTags,
		}
		volumeID, err = v.d.client.CreateVolume(r)

		if err != nil {
			return nil, err
		}

		log.Debugf("Created volume %v from EBS snapshot %v", id, ebsSnapshotID)
	} else {

		// CREATE a NEW EBS volume
		volumeSize, err = v.d.getSize(opts, v.d.DefaultVolumeSize)
		if err != nil {
			return nil, err
		}

		volumeType, iops, err := v.d.getTypeAndIOPS(opts)
		if err != nil {
			return nil, err
		}

		r := &CreateEBSVolumeRequest{
			Size:       volumeSize,
			VolumeType: volumeType,
			IOPS:       iops,
			Tags:       newTags,
			KmsKeyID:   v.d.DefaultKmsKeyID,
		}

		volumeID, err = v.d.client.CreateVolume(r)
		if err != nil {
			return nil, err
		}

		log.Debugf("Created volume %s from EBS volume %v", id, volumeID)
		// This is true only for NEWLY created volumes.
		format = true
	}

	dev, err := v.d.client.AttachVolume(volumeID, volumeSize)
	if err != nil {
		return nil, err
	}

	log.Debugf("Attached EBS volume: %v to device: %v", volumeID, dev)

	volume.Name = id
	volume.EBSID = volumeID
	volume.Device = dev
	volume.Snapshots = make(map[string]Snapshot)

	// Do NOT format EXISTING or snapshot RESTORED volume
	if format {
		if _, err := util.Execute("mkfs", []string{"-t", "ext4", dev}); err != nil {
			return nil, err
		}
	}

	return &driver.Response{}, util.ObjectSave(volume)
}

// Package ebs provides a concrete implementation of Mtest driver.
package ebs

import (
	"fmt"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	. "github.com/openebs/mtest/logging"
)

const (
	// Name of this Mtest Driver implementation
	// The name that this driver will be known by the outside world
	DRIVER_NAME = "ebs"

	// Configuration file of this Mtest Driver implementation
	DRIVER_CONFIG_FILE = "ebs.cfg"

	// CFG_PREFIX is used to locate the Volume's config path
	CFG_PREFIX = DRIVER_NAME + "_"

	// VOLUME_CFG_PREFIX is used to locate the Volume's config path
	VOLUME_CFG_PREFIX = "volume_"

	// CFG_POSTFIX is used to locate the Volume's config path
	CFG_POSTFIX = ".json"

	// Default volume size property used by ebs driver
	EBS_DEFAULT_VOLUME_SIZE = "ebs.defaultvolumesize"

	// Default volume type property used by ebs driver
	EBS_DEFAULT_VOLUME_TYPE = "ebs.defaultvolumetype"

	// Default volume key property used by ebs driver
	EBS_DEFAULT_VOLUME_KEY = "ebs.defaultkmskeyid"

	// Default volume size used by ebs driver
	DEFAULT_VOLUME_SIZE = "4G"

	// Default volume type used by ebs driver
	DEFAULT_VOLUME_TYPE = "gp2"

	// The mount directory used by ebs driver
	MOUNTS_DIR = "mounts"

	// The binary used by ebs driver for mount related operations
	MOUNT_BINARY = "mount"

	// The binary used by ebs driver for unmount related operations
	UMOUNT_BINARY = "umount"

	// Mount point parameter
	OPT_MOUNT_POINT = "MountPoint"

	// Volume size parameter
	OPT_SIZE = "Size"

	// Volume format parameter
	OPT_FORMAT = "Format"

	// Volume ID parameter
	OPT_VOLUME_ID = "VolumeID"

	// Volume Name parameter
	OPT_VOLUME_NAME = "VolumeName"

	// Volume Driver parameter
	OPT_VOLUME_DRIVER_ID = "VolumeDriverID"

	// Volume Type parameter
	OPT_VOLUME_TYPE = "VolumeType"

	// Volume IOPS parameter
	OPT_VOLUME_IOPS = "VolumeIOPS"

	// Volume Created Time parameter
	OPT_VOLUME_CREATED_TIME = "VolumeCreatedAt"

	// Snapshot ID parameter
	OPT_SNAPSHOT_ID = "SnapshotID"

	// Snapshot Name parameter
	OPT_SNAPSHOT_NAME = "SnapshotName"

	// Snapshot Created Time parameter
	OPT_SNAPSHOT_CREATED_TIME = "SnapshotCreatedAt"

	// Backup URL parameter
	OPT_BACKUP_URL = "BackupURL"

	// Reference Only parameter
	OPT_REFERENCE_ONLY = "ReferenceOnly"

	// Prepare for VM parameter
	OPT_PREPARE_FOR_VM = "PrepareForVM"

	// Filesystem parameter
	OPT_FILESYSTEM = "Filesystem"
)

var (
	log = logrus.WithFields(logrus.Fields{"pkg": "mtest.driver.ebs"})
)

// A Device represent the core storage properties.
// Most of the volume defaults are defined here.
type Device struct {
	Root              string
	DefaultVolumeSize int64
	DefaultVolumeType string
	DefaultKmsKeyID   string
}

// Get the device config path
func (dev *Device) ConfigFile() (string, error) {
	if dev.Root == "" {
		return "", fmt.Errorf("BUG: Invalid empty device config path")
	}
	return filepath.Join(dev.Root, DRIVER_CONFIG_FILE), nil
}

// A Snapshot provides a structure to represent a snapshot on
// a particular volume.
type Snapshot struct {
	Name       string
	VolumeName string
	EBSID      string
}

// A Volume provides a structure to identify & define an ebs storage.
type Volume struct {
	Name       string
	EBSID      string
	Device     string
	MountPoint string
	Snapshots  map[string]Snapshot

	configPath string
}

// Get the volume's config file from a pre-determined location
// i.e. the location makes use of volume-config-path, driver-name &
// volume-name among other things.
func (v *Volume) ConfigFile() (string, error) {
	if v.Name == "" {
		return "", fmt.Errorf("BUG: Invalid empty volume name")
	}
	if v.configPath == "" {
		return "", fmt.Errorf("BUG: Invalid empty volume config path")
	}
	return filepath.Join(v.configPath, CFG_PREFIX+VOLUME_CFG_PREFIX+v.Name+CFG_POSTFIX), nil
}

// Get the volume's device
func (v *Volume) GetDevice() (string, error) {
	return v.Device, nil
}

// Get various mount options for the volume
func (v *Volume) GetMountOpts() []string {
	return []string{}
}

// Get the default mount point of the volume. This default makes use
// of volume-config-path & volume-name among other things.
func (v *Volume) GenerateDefaultMountPoint() string {
	return filepath.Join(v.configPath, MOUNTS_DIR, v.Name)
}

// Generate error with ebs label
func generateError(fields logrus.Fields, format string, v ...interface{}) error {
	return ErrorWithFields("ebs", fields, format, v)
}

// Verify the volume types to be EBS compliant
func checkVolumeType(volumeType string) error {
	validVolumeType := map[string]bool{
		"gp2":      true,
		"io1":      true,
		"standard": true,
		"st1":      true,
		"sc1":      true,
	}
	if !validVolumeType[volumeType] {
		return fmt.Errorf("Invalid volume type %v", volumeType)
	}
	return nil
}

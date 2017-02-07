package ebs

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/openebs/mtest/driver"
	. "github.com/openebs/mtest/logging"
	"github.com/openebs/mtest/util"
)

// Properties used by ebs driver for various
// volume related operations. These are typically
// set as keys in a map of options passed around
// during execution of volume operations.
const (
	OPT_MOUNT_POINT           = "MountPoint"
	OPT_SIZE                  = "Size"
	OPT_FORMAT                = "Format"
	OPT_VOLUME_NAME           = "VolumeName"
	OPT_VOLUME_DRIVER_ID      = "VolumeDriverID"
	OPT_VOLUME_TYPE           = "VolumeType"
	OPT_VOLUME_IOPS           = "VolumeIOPS"
	OPT_VOLUME_CREATED_TIME   = "VolumeCreatedAt"
	OPT_SNAPSHOT_NAME         = "SnapshotName"
	OPT_SNAPSHOT_CREATED_TIME = "SnapshotCreatedAt"
	OPT_BACKUP_URL            = "BackupURL"
	OPT_REFERENCE_ONLY        = "ReferenceOnly"
	OPT_PREPARE_FOR_VM        = "PrepareForVM"
	OPT_FILESYSTEM            = "Filesystem"
)

// InitExecFunc is the initialize function for each EBS driver executor(s).
// Each executor must implement this function and register itself
// through RegisterAsEBSExecutor().
type InitExecFunc func(d *EBSDriver) (driver.Executor, error)

var (
	executors map[string]InitExecFunc
)

// EBSDriver manages the volume related operations
// This is one of the concrete Mtest driver implementations.
type EBSDriver struct {
	mutex  *sync.RWMutex
	client *ebsClient
	Device
}

func init() {
	// Register by passing the name of the driver
	// and the function definition.
	driver.Register(DRIVER_NAME, Init)
	executors = make(map[string]InitExecFunc)
}

// Initialize the EBSDriver as a MtestDriver
func Init(root string, config map[string]string) (driver.MtestDriver, error) {
	ebsClient, err := NewEBSClient()
	if err != nil {
		return nil, err
	}

	dev := &Device{
		Root: root,
	}

	exists, err := util.ObjectExists(dev)
	if err != nil {
		return nil, err
	}

	if exists {
		if err := util.ObjectLoad(dev); err != nil {
			return nil, err
		}
	} else {
		if err := util.MkdirIfNotExists(root); err != nil {
			return nil, err
		}

		if config[EBS_DEFAULT_VOLUME_SIZE] == "" {
			config[EBS_DEFAULT_VOLUME_SIZE] = DEFAULT_VOLUME_SIZE
		}

		size, err := util.ParseSize(config[EBS_DEFAULT_VOLUME_SIZE])
		if err != nil {
			return nil, err
		}

		if config[EBS_DEFAULT_VOLUME_TYPE] == "" {
			config[EBS_DEFAULT_VOLUME_TYPE] = DEFAULT_VOLUME_TYPE
		}

		volumeType := config[EBS_DEFAULT_VOLUME_TYPE]
		if err := checkVolumeType(volumeType); err != nil {
			return nil, err
		}

		kmsKeyId := config[EBS_DEFAULT_VOLUME_KEY]

		dev = &Device{
			Root:              root,
			DefaultVolumeSize: size,
			DefaultVolumeType: volumeType,
			DefaultKmsKeyID:   kmsKeyId,
		}

		if err := util.ObjectSave(dev); err != nil {
			return nil, err
		}
	}

	d := &EBSDriver{
		mutex:  &sync.RWMutex{},
		client: ebsClient,
		Device: *dev,
	}

	if err := d.remountVolumes(); err != nil {
		return nil, err
	}

	return d, nil
}

// Register executors, i.e. add implemented executors via `InitExecFunc`
// of the known driver. `InitExecFunc` is supposed to be defined in
// each driver executor implementation.
func RegisterAsEBSExecutor(name string, iExecFn InitExecFunc) error {
	_, exists := executors[name]

	if exists {
		return fmt.Errorf("Executor: %s already registered with driver: %s", name, DRIVER_NAME)
	}

	executors[name] = iExecFn
	return nil
}

// Fetch a list of Executors based on the provided hints
func (d *EBSDriver) Executors(hints ...string) (map[string]driver.Executor, error) {
	// define a map of executors
	execs := make(map[string]driver.Executor)

	for _, hint := range hints {
		_, exists := executors[hint]

		if !exists {
			log.Warnf("Executor not initialized for: %s", hint)
			continue
		}

		// Invoke the executor initialization function
		// by providing the hint & the driver instance
		executor, err := executors[hint](d)

		if err != nil {
			log.Warnf(" Failed to fetch executor %s, err: %v", hint, err)
			continue
		}

		execs[hint] = executor
	}

	if len(execs) == 0 {
		return nil, fmt.Errorf("No executors found with hints: %v", hints)
	}

	return execs, nil
}

// Get a volume with device's root as the volume's path
func (d *EBSDriver) blankVolume(name string) *Volume {
	return &Volume{
		configPath: d.Root,
		Name:       name,
	}
}

// Remount all the volumes associated with this
// EBSDriver
func (d *EBSDriver) remountVolumes() error {
	volumeIDs, err := d.listVolumeNames()
	if err != nil {
		return err
	}
	for _, id := range volumeIDs {
		volume := d.blankVolume(id)
		if err := util.ObjectLoad(volume); err != nil {
			return err
		}
		if volume.MountPoint == "" {
			continue
		}
		req := driver.Request{
			Name:    id,
			Options: map[string]string{},
		}
		if _, err := d.MountVolume(req); err != nil {
			return err
		}
	}
	return err
}

// Get the name of this Mtest Driver.
func (d *EBSDriver) Name() string {
	return DRIVER_NAME
}

// Get extra info with respect to this Mtest Driver.
func (d *EBSDriver) Info() (map[string]string, error) {
	infos := make(map[string]string)
	infos["DefaultVolumeSize"] = strconv.FormatInt(d.DefaultVolumeSize, 10)
	infos["DefaultVolumeType"] = d.DefaultVolumeType
	infos["DefaultKmsKey"] = d.DefaultKmsKeyID
	infos["InstanceID"] = d.client.InstanceID
	infos["Region"] = d.client.Region
	infos["AvailiablityZone"] = d.client.AvailabilityZone
	return infos, nil
}

func (d *EBSDriver) getSize(opts map[string]string, defaultVolumeSize int64) (int64, error) {
	size := opts[OPT_SIZE]
	if size == "" || size == "0" {
		size = strconv.FormatInt(defaultVolumeSize, 10)
	}

	return util.ParseSize(size)
}

func (d *EBSDriver) getTypeAndIOPS(opts map[string]string) (string, int64, error) {
	var (
		iops int64
		err  error
	)

	volumeType := opts[OPT_VOLUME_TYPE]
	if volumeType == "" {
		volumeType = d.DefaultVolumeType
	}

	if err := checkVolumeType(volumeType); err != nil {
		return "", 0, err
	}

	if opts[OPT_VOLUME_IOPS] != "" {
		iops, err = strconv.ParseInt(opts[OPT_VOLUME_IOPS], 10, 64)
		if err != nil {
			return "", 0, err
		}
	}

	if volumeType == "io1" && iops == 0 {
		return "", 0, fmt.Errorf("Invalid IOPS for volume type io1")
	}

	if volumeType != "io1" && iops != 0 {
		return "", 0, fmt.Errorf("IOPS only valid for volume type io1")
	}

	return volumeType, iops, nil
}

func (d *EBSDriver) MountVolume(req driver.Request) (string, error) {
	id := req.Name
	opts := req.Options

	volume := d.blankVolume(id)
	if err := util.ObjectLoad(volume); err != nil {
		return "", err
	}

	mountPoint, err := util.VolumeMount(volume, opts[OPT_MOUNT_POINT], false)
	if err != nil {
		return "", err
	}

	if err := util.ObjectSave(volume); err != nil {
		return "", err
	}

	return mountPoint, nil
}

func (d *EBSDriver) UmountVolume(req driver.Request) error {
	id := req.Name

	volume := d.blankVolume(id)
	if err := util.ObjectLoad(volume); err != nil {
		return err
	}

	if err := util.VolumeUmount(volume); err != nil {
		return err
	}

	if err := util.ObjectSave(volume); err != nil {
		return err
	}

	return nil
}

func (d *EBSDriver) MountPoint(req driver.Request) (string, error) {
	id := req.Name

	volume := d.blankVolume(id)
	if err := util.ObjectLoad(volume); err != nil {
		return "", err
	}
	return volume.MountPoint, nil
}

func (d *EBSDriver) listVolumeNames() ([]string, error) {
	return util.ListConfigIDs(d.Root, CFG_PREFIX+VOLUME_CFG_PREFIX, CFG_POSTFIX)
}

func (d *EBSDriver) getSnapshotAndVolume(snapshotID, volumeID string) (*Snapshot, *Volume, error) {
	volume := d.blankVolume(volumeID)

	if err := util.ObjectLoad(volume); err != nil {
		return nil, nil, err
	}

	snap, exists := volume.Snapshots[snapshotID]
	if !exists {
		return nil, nil, generateError(logrus.Fields{
			LOG_FIELD_VOLUME:   volumeID,
			LOG_FIELD_SNAPSHOT: snapshotID,
		}, "cannot find snapshot of volume")
	}
	return &snap, volume, nil
}

func (d *EBSDriver) GetSnapshotInfo(req driver.Request) (map[string]string, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	id := req.Name
	volumeID, err := util.GetFieldFromOpts(OPT_VOLUME_NAME, req.Options)
	if err != nil {
		return nil, err
	}

	return d.getSnapshotInfo(id, volumeID)
}

func (d *EBSDriver) getSnapshotInfo(id, volumeID string) (map[string]string, error) {
	// Snapshot on EBS can be removed by DeleteBackup
	removed := false

	snapshot, _, err := d.getSnapshotAndVolume(id, volumeID)
	if err != nil {
		return nil, err
	}

	ebsSnapshot, err := d.client.GetSnapshot(snapshot.EBSID)
	if err != nil {
		removed = true
	}

	info := map[string]string{}
	if !removed {
		info = map[string]string{
			OPT_SNAPSHOT_NAME:         snapshot.Name,
			"VolumeName":              volumeID,
			"EBSSnapshotID":           aws.StringValue(ebsSnapshot.SnapshotId),
			"EBSVolumeID":             aws.StringValue(ebsSnapshot.VolumeId),
			"KmsKeyId":                aws.StringValue(ebsSnapshot.KmsKeyId),
			OPT_SNAPSHOT_CREATED_TIME: (*ebsSnapshot.StartTime).Format(time.RubyDate),
			OPT_SIZE:                  strconv.FormatInt(*ebsSnapshot.VolumeSize*GB, 10),
			"State":                   aws.StringValue(ebsSnapshot.State),
		}
	} else {
		info = map[string]string{
			OPT_SNAPSHOT_NAME: snapshot.Name,
			"VolumeName":      volumeID,
			"State":           "removed",
		}
	}

	return info, nil
}

func (d *EBSDriver) ListSnapshot(opts map[string]string) (map[string]map[string]string, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	var (
		volumeIDs []string
		err       error
	)

	snapshots := make(map[string]map[string]string)
	specifiedVolumeID, _ := util.GetFieldFromOpts(OPT_VOLUME_NAME, opts)

	if specifiedVolumeID != "" {
		volumeIDs = []string{
			specifiedVolumeID,
		}
	} else {
		volumeIDs, err = d.listVolumeNames()
		if err != nil {
			return nil, err
		}
	}

	for _, volumeID := range volumeIDs {
		volume := d.blankVolume(volumeID)
		if err := util.ObjectLoad(volume); err != nil {
			return nil, err
		}
		for snapshotID := range volume.Snapshots {
			snapshots[snapshotID], err = d.getSnapshotInfo(snapshotID, volumeID)
			if err != nil {
				return nil, err
			}
		}
	}

	return snapshots, nil
}

func checkEBSSnapshotID(id string) error {
	validID := regexp.MustCompile(`^snap-[0-9a-z]+$`)
	if !validID.MatchString(id) {
		return fmt.Errorf("Invalid EBS snapshot id %v", id)
	}
	return nil
}

func encodeURL(region, ebsSnapshotID string) string {
	return "ebs://" + region + "/" + ebsSnapshotID
}

func decodeURL(backupURL string) (string, string, error) {
	u, err := url.Parse(backupURL)
	if err != nil {
		return "", "", err
	}
	if u.Scheme != DRIVER_NAME {
		return "", "", fmt.Errorf("BUG: Why dispatch %v to %v?", u.Scheme, DRIVER_NAME)
	}

	region := u.Host
	ebsSnapshotID := strings.TrimRight(strings.TrimLeft(u.Path, "/"), "/")
	if err := checkEBSSnapshotID(ebsSnapshotID); err != nil {
		return "", "", err
	}

	return region, ebsSnapshotID, nil
}

func (d *EBSDriver) CreateBackup(snapshotID, volumeID, destURL string, opts map[string]string) (string, error) {
	//destURL is not necessary in EBS case
	snapshot, _, err := d.getSnapshotAndVolume(snapshotID, volumeID)
	if err != nil {
		return "", err
	}

	if err := d.client.WaitForSnapshotComplete(snapshot.EBSID); err != nil {
		return "", err
	}
	return encodeURL(d.client.Region, snapshot.EBSID), nil
}

func (d *EBSDriver) DeleteBackup(backupURL string) error {
	// Would remove the snapshot
	region, ebsSnapshotID, err := decodeURL(backupURL)
	if err != nil {
		return err
	}
	if err := d.client.DeleteSnapshotWithRegion(ebsSnapshotID, region); err != nil {
		return err
	}
	return nil
}

func (d *EBSDriver) GetBackupInfo(backupURL string) (map[string]string, error) {
	region, ebsSnapshotID, err := decodeURL(backupURL)
	if err != nil {
		return nil, err
	}

	ebsSnapshot, err := d.client.GetSnapshotWithRegion(ebsSnapshotID, region)
	if err != nil {
		return nil, err
	}

	info := map[string]string{
		"Region":        region,
		"EBSSnapshotID": aws.StringValue(ebsSnapshot.SnapshotId),
		"EBSVolumeID":   aws.StringValue(ebsSnapshot.VolumeId),
		"KmsKeyId":      aws.StringValue(ebsSnapshot.KmsKeyId),
		"StartTime":     (*ebsSnapshot.StartTime).Format(time.RubyDate),
		"Size":          strconv.FormatInt(*ebsSnapshot.VolumeSize*GB, 10),
		"State":         aws.StringValue(ebsSnapshot.State),
	}

	return info, nil
}

func (d *EBSDriver) ListBackup(destURL string, opts map[string]string) (map[string]map[string]string, error) {
	//EBS doesn't support ListBackup(), return empty to satisfy caller
	return map[string]map[string]string{}, nil
}

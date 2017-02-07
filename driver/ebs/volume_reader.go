package ebs

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/openebs/mtest/driver"
	"github.com/openebs/mtest/util"
)

const (
	// Name of this executor
	// This executor will be known as this to the outside world
	EBS_VOLUME_READ_EXEC = "ebs.volume.read.executor"
)

// This is a EBS driver executor.
type VolumeReader struct {
	d *EBSDriver
}

func init() {
	// Register by passing the name of this executor
	// and its initializing function definition.
	RegisterAsEBSExecutor(EBS_VOLUME_READ_EXEC, VolumeReaderInit)
}

// The initializing function of VolumeReader executor.
func VolumeReaderInit(ebsDriver *EBSDriver) (driver.Executor, error) {
	return &VolumeReader{
		d: ebsDriver,
	}, nil
}

func (v *VolumeReader) Exec(req driver.Request) (*driver.Response, error) {

	v.d.mutex.Lock()
	defer v.d.mutex.Unlock()

	_, exists := req.Options["uuid"]
	if !exists {
		return nil, fmt.Errorf("Volume id not provided")
	}

	volume := v.d.blankVolume(req.Options["uuid"])
	if err := util.ObjectLoad(volume); err != nil {
		return nil, err
	}

	ebsVolume, err := v.d.client.GetVolume(volume.EBSID)
	if err != nil {
		return nil, err
	}

	iops := ""
	if ebsVolume.Iops != nil {
		iops = strconv.FormatInt(*ebsVolume.Iops, 10)
	}

	info := map[string]interface{}{
		"Device":                volume.Device,
		"MountPoint":            volume.MountPoint,
		"EBSVolumeID":           volume.EBSID,
		"KmsKeyId":              aws.StringValue(ebsVolume.KmsKeyId),
		"AvailiablityZone":      aws.StringValue(ebsVolume.AvailabilityZone),
		OPT_VOLUME_NAME:         req.Options["uuid"],
		OPT_VOLUME_CREATED_TIME: (*ebsVolume.CreateTime).Format(time.RubyDate),
		"Size":                  strconv.FormatInt(*ebsVolume.Size*GB, 10),
		"State":                 aws.StringValue(ebsVolume.State),
		"Type":                  aws.StringValue(ebsVolume.VolumeType),
		"IOPS":                  iops,
	}

	return &driver.Response{
		Values: info,
	}, nil
}

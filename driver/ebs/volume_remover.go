package ebs

import (
	"strconv"

	"github.com/openebs/mtest/driver"
	"github.com/openebs/mtest/util"
)

const (
	// Name of this executor
	// This executor will be known as this to the outside world
	EBS_VOLUME_REMOVE_EXEC = "ebs.volume.remove.executor"
)

// This is a EBS driver executor.
type VolumeRemover struct {
	d *EBSDriver
}

func init() {
	// Register by passing the name of this executor
	// and its initializing function definition.
	RegisterAsEBSExecutor(EBS_VOLUME_REMOVE_EXEC, VolumeRemoverInit)
}

// The initializing function of VolumeRemover executor.
func VolumeRemoverInit(ebsDriver *EBSDriver) (driver.Executor, error) {
	return &VolumeRemover{
		d: ebsDriver,
	}, nil
}

func (v *VolumeRemover) Exec(req driver.Request) (*driver.Response, error) {

	v.d.mutex.Lock()
	defer v.d.mutex.Unlock()

	id := req.Name
	opts := req.Options

	volume := v.d.blankVolume(id)
	err := util.ObjectLoad(volume)
	if err != nil {
		return nil, err
	}

	referenceOnly, _ := strconv.ParseBool(opts[OPT_REFERENCE_ONLY])

	err = v.d.client.DetachVolume(volume.EBSID)
	if err != nil {
		if !referenceOnly {
			return nil, err
		}

		//Ignore the error, remove the reference
		log.Warnf("Unable to detach %v(%v) due to %v, but continue with removing the reference",
			id, volume.EBSID, err)

	} else {
		log.Debugf("Detached %v(%v) from %v", id, volume.EBSID, volume.Device)
	}

	if !referenceOnly {
		err := v.d.client.DeleteVolume(volume.EBSID)
		if err != nil {
			return nil, err
		}

		log.Debugf("Deleted volume %v(%v)", id, volume.EBSID)
	}

	return &driver.Response{}, util.ObjectDelete(volume)
}

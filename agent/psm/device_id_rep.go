package psm

import "github.com/optechlab/findy-go/dto"

type DeviceIDRep struct {
	DID         string
	DeviceToken string
}

func NewDeviceIDRep(d []byte) *DeviceIDRep {
	p := &DeviceIDRep{}
	dto.FromGOB(d, p)
	return p
}

func (p *DeviceIDRep) Data() []byte {
	return dto.ToGOB(p)
}

func (p *DeviceIDRep) Key() []byte {
	return []byte(p.DID + p.DeviceToken)
}

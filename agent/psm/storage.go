package psm

import (
	"github.com/optechlab/findy-agent/agent/endp"
)

var (
	lmDb = &DB{}
)

func Open(filename string) (err error) {
	return lmDb.Open(filename)
}

func AddRawPL(addr *endp.Addr, data []byte) (err error) {
	return lmDb.AddRawPL(addr, data)
}

func RmRawPL(addr *endp.Addr) (err error) {
	return lmDb.RmRawPL(addr)
}

func GetPSM(key StateKey) (s *PSM, err error) {
	return lmDb.GetPSM(key)
}

func AddPSM(p *PSM) (err error) {
	return lmDb.addPSM(p)
}

func IsPSMReady(key StateKey) (yes bool, err error) {
	return lmDb.IsPSMReady(key)
}

func AllPSM(did string, tsSince *int64) (m *[]PSM, err error) {
	return lmDb.AllPSM(did, tsSince)
}

func AddPairwiseRep(p *PairwiseRep) (err error) {
	return lmDb.AddPairwiseRep(p)
}

func GetPairwiseRep(k StateKey) (m *PairwiseRep, err error) {
	return lmDb.GetPairwiseRep(k)
}

func AddDeviceIDRep(d *DeviceIDRep) (err error) {
	return lmDb.AddDeviceIDRep(d)
}

func GetAllDeviceIDRep(did string) (m *[]DeviceIDRep, err error) {
	return lmDb.GetAllDeviceIDRep(did)
}

func AddBasicMessageRep(p *BasicMessageRep) (err error) {
	return lmDb.AddBasicMessageRep(p)
}

func GetBasicMessageRep(k StateKey) (m *BasicMessageRep, err error) {
	return lmDb.GetBasicMessageRep(k)
}

func AddIssueCredRep(p *IssueCredRep) (err error) {
	return lmDb.AddIssueCredRep(p)
}

func GetIssueCredRep(k StateKey) (m *IssueCredRep, err error) {
	return lmDb.GetIssueCredRep(k)
}

func AddPresentProofRep(p *PresentProofRep) (err error) {
	return lmDb.AddPresentProofRep(p)
}

func GetPresentProofRep(k StateKey) (m *PresentProofRep, err error) {
	return lmDb.GetPresentProofRep(k)
}

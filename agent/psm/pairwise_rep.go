package psm

import (
	"github.com/findy-network/findy-wrapper-go/dto"
)

type PairwiseRep struct {
	Key        StateKey
	Name       string
	TheirLabel string
	Caller     DIDRep
	Callee     DIDRep
}

func NewPairwiseRep(d []byte) *PairwiseRep {
	p := &PairwiseRep{}
	dto.FromGOB(d, p)
	return p
}

func (p *PairwiseRep) Data() []byte {
	return dto.ToGOB(p)
}

func (p *PairwiseRep) KData() []byte {
	return p.Key.Data()
}

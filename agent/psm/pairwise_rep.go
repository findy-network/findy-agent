package psm

import (
	"github.com/findy-network/findy-wrapper-go/dto"
)

type PairwiseRep struct {
	Name       string // In our implementation this is connection id!
	Key        StateKey
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

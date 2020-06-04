package psm

import (
	"github.com/optechlab/findy-agent/agent/ssi"
	"github.com/optechlab/findy-go/dto"
)

type WalletRep struct {
	DID string
	ssi.Wallet
}

func NewWalletRep(d []byte) *WalletRep {
	p := &WalletRep{}
	dto.FromGOB(d, p)
	return p
}

func (p *WalletRep) Data() []byte {
	return dto.ToGOB(p)
}

func (p *WalletRep) Key() []byte {
	return []byte(p.DID)
}

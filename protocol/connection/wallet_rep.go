package connection

import (
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-wrapper-go/dto"
)

type walletRep struct {
	DID string
	ssi.Wallet
}

func NewWalletRep(d []byte) *walletRep {
	p := &walletRep{}
	dto.FromGOB(d, p)
	return p
}

func (p *walletRep) Data() []byte {
	return dto.ToGOB(p)
}

func (p *walletRep) Key() []byte {
	return []byte(p.DID)
}

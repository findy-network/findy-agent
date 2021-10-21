package connection

import (
	"github.com/findy-network/findy-agent/agent/ssi"
)

type walletRep struct {
	DID string
	ssi.Wallet
}

package vdr

import (
	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/hyperledger/aries-framework-go/pkg/framework/aries/api/vdr"
	vdregistry "github.com/hyperledger/aries-framework-go/pkg/vdr"
	"github.com/hyperledger/aries-framework-go/pkg/vdr/key"
	"github.com/hyperledger/aries-framework-go/pkg/vdr/peer"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

type VDR struct {
	registry vdr.Registry

	keyVDR  vdr.VDR
	peerVDR vdr.VDR
}

type Config struct {
	Key      string
	FileName string
	FilePath string
}

func New(storage api.AgentStorage) (v *VDR, err error) {
	defer err2.Returnf(&err, "vdr new")
	v = &VDR{
		keyVDR: &key.VDR{},
	}
	peerVDR := try.To1(peer.New(storage))

	v.peerVDR = peerVDR

	v.registry = vdregistry.New(
		vdregistry.WithVDR(v.keyVDR),
		vdregistry.WithVDR(v.peerVDR),
	)

	return v, nil
}

func (v *VDR) Key() vdr.VDR {
	return v.keyVDR
}

func (v *VDR) Peer() vdr.VDR {
	return v.peerVDR
}

func (v *VDR) Registry() vdr.Registry {
	return v.registry
}

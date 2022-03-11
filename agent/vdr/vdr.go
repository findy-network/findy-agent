package vdr

import (
	"github.com/findy-network/findy-agent/agent/storage/mgddb"
	"github.com/hyperledger/aries-framework-go/pkg/framework/aries/api/vdr"
	registry "github.com/hyperledger/aries-framework-go/pkg/vdr"
	"github.com/hyperledger/aries-framework-go/pkg/vdr/key"
	"github.com/hyperledger/aries-framework-go/pkg/vdr/peer"
	"github.com/lainio/err2"
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

func New(storage *mgddb.Storage) (v *VDR, err error) {
	defer err2.Annotate("vdr new", &err)
	v = &VDR{
		keyVDR: &key.VDR{},
	}
	peerVDR, err := peer.New(storage)
	err2.Check(err)

	v.peerVDR = peerVDR

	v.registry = registry.New(
		registry.WithVDR(v.keyVDR),
		registry.WithVDR(v.peerVDR),
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

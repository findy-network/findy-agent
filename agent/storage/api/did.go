package api

import (
	"strings"

	"github.com/golang/glog"
	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
)

type DIDMethod string

const (
	DIDMethodPrefix                = "did:"
	DIDMethodKey         DIDMethod = DIDMethodPrefix + "key"
	DIDMethodPeer        DIDMethod = DIDMethodPrefix + "peer"
	DIDMethodIndy        DIDMethod = DIDMethodPrefix + "indy"
	DIDMethodWeb         DIDMethod = DIDMethodPrefix + "web"
	DIDMethodUnsupported DIDMethod = "unsupported"
)

type DID struct {
	ID         string // ID is the key. Use real DID value for that
	DID        string
	KID        string
	IndyVerKey string
	Doc        *did.DocResolution // TODO: bring our own core.DIDDoc
}

// just playing around, this is probably not needed at this level
func (d *DID) Method() DIDMethod {
	methods := []DIDMethod{
		DIDMethodKey, DIDMethodPeer, DIDMethodIndy, DIDMethodWeb,
	}
	for _, method := range methods {
		if strings.HasPrefix(d.DID, string(method)) {
			return method
		}
	}
	glog.Warningf("DID method not found for %s", d.DID)
	return DIDMethodUnsupported
}

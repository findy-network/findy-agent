package didexchange

import (
	"strings"

	"github.com/findy-network/findy-agent/agent/sec"
	"github.com/findy-network/findy-agent/agent/service"
)

func (r *Response) Sign(pipe sec.Pipe) (err error) {
	r.ConnectionSignature, err = r.Connection.buildConnectionSignature(pipe)
	return err
}

func (r *Response) Verify() (ok bool, err error) {
	r.Connection, err = r.ConnectionSignature.verifySignature(nil)
	ok = r.Connection != nil

	if ok {
		rawDID := r.Connection.DIDDoc.ID
		r.Connection.DID = strings.TrimPrefix(rawDID, "did:sov:")
	}

	return ok, err
}

func (r *Response) Endpoint() service.Addr {
	if len(r.Connection.DIDDoc.Service) == 0 {
		return service.Addr{}
	}

	addr := r.Connection.DIDDoc.Service[0].ServiceEndpoint
	key := r.Connection.DIDDoc.Service[0].RecipientKeys[0]

	return service.Addr{Endp: addr, Key: key}
}

func (r *Response) SetEndpoint(ae service.Addr) {
	if len(r.Connection.DIDDoc.Service) == 0 {
		panic("we should not be here")
	}

	r.Connection.DIDDoc.Service[0].ServiceEndpoint = ae.Endp
	r.Connection.DIDDoc.Service[0].RecipientKeys[0] = ae.Key
}

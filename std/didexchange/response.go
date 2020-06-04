package didexchange

import (
	"strings"

	"github.com/findy-network/findy-agent/agent/sec"
	"github.com/findy-network/findy-agent/agent/service"
)

func (r *Response) Sign(pipe sec.Pipe) (err error) {
	r.ConnectionSignature, err = r.connection.buildConnectionSignature(pipe)
	return err
}

func (r *Response) Verify() (ok bool, err error) {
	r.connection, err = r.ConnectionSignature.verifySignature(nil)
	ok = r.connection != nil

	if ok {
		rawDID := r.connection.DIDDoc.ID
		r.connection.DID = strings.TrimPrefix(rawDID, "did:sov:")
	}

	return ok, err
}

func (r *Response) Endpoint() service.Addr {
	if len(r.connection.DIDDoc.Service) == 0 {
		return service.Addr{}
	}

	addr := r.connection.DIDDoc.Service[0].ServiceEndpoint
	key := r.connection.DIDDoc.Service[0].RecipientKeys[0]

	return service.Addr{Endp: addr, Key: key}
}

func (r *Response) SetEndpoint(ae service.Addr) {
	if len(r.connection.DIDDoc.Service) == 0 {
		panic("we should not be here")
	}

	r.connection.DIDDoc.Service[0].ServiceEndpoint = ae.Endp
	r.connection.DIDDoc.Service[0].RecipientKeys[0] = ae.Key
}

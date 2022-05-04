package didexchange

import (
	"github.com/findy-network/findy-agent/agent/service"
)

func (r *Response) Endpoint() service.Addr {
	if len(r.Connection.Doc.Service) == 0 {
		return service.Addr{}
	}

	addr := r.Connection.Doc.Service[0].ServiceEndpoint
	key := r.Connection.Doc.Service[0].RecipientKeys[0]

	return service.Addr{Endp: addr, Key: key}
}

func (r *Response) SetEndpoint(ae service.Addr) {
	if len(r.Connection.Doc.Service) == 0 {
		panic("we should not be here")
	}

	r.Connection.Doc.Service[0].ServiceEndpoint = ae.Endp
	r.Connection.Doc.Service[0].RecipientKeys[0] = ae.Key
}

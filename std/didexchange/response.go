package didexchange

import (
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/std/common"
)

func (r *Response) Endpoint() service.Addr {
	services := common.Services(r.Connection.DIDDoc)
	if len(services) == 0 {
		return service.Addr{}
	}

	addr := services[0].ServiceEndpoint
	key := services[0].RecipientKeys[0]

	return service.Addr{Endp: addr, Key: key}
}

func (r *Response) SetEndpoint(ae service.Addr) {
	services := common.Services(r.Connection.DIDDoc)
	if len(services) == 0 {
		panic("we should not be here")
	}

	services[0].ServiceEndpoint = ae.Endp
	services[0].RecipientKeys[0] = ae.Key

	common.SetServices(r.Connection.DIDDoc, services)
}

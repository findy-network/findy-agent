package didexchange

import (
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/std/common"
)

func (r *Response) Endpoint() (service.Addr, error) {
	services := common.Services(r.Connection.DIDDoc)
	if len(services) == 0 {
		return service.Addr{}, nil
	}

	addr := services[0].ServiceEndpoint
	key := services[0].RecipientKeys[0]

	return service.Addr{Endp: addr, Key: key}, nil
}

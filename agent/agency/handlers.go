/*
Package agency offers mainly internal services for Agency framework to help
implement multi tenant agent service.

Please note that some of the exported methods and variables aren't important for
framework user. They are exposed because the refactoring was done in hurry and
we didn't have time to move some of them to internal package.
*/
package agency

import (
	"sync"

	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/txp"
	"github.com/golang/glog"
)

const (
	APIPath      = "api"    // Agency API path, cannot set yet
	CAAPIPath    = "ca-api" // default for CA API, serviceName
	ProtocolPath = "a2a"    // default for A2A Protocols (Aries), serviceName2
)

type Endpoint = string

var (
	// handlers are the http endpoint handlers for CA API and protocols. This
	// is a dynamic map which includes all of the active CAs. On-boarded CAs
	// are but directly to this map. After restart of the agency the map is
	// empty and all of the CA endpoints are in the seedHandlers.
	handlers = struct {
		sync.RWMutex
		m map[Endpoint]comm.Handler
	}{
		m: make(map[Endpoint]comm.Handler),
	}

	// seedHandlers is the map of the seed endpoints which are not yet fully
	// constructed CAs. After first endpoint access they are moved to handlers
	// map and the corresponding CA is constructed.
	seedHandlers = struct {
		sync.RWMutex
		m map[Endpoint]comm.SeedHandler
	}{
		m: make(map[Endpoint]comm.SeedHandler),
	}

	agencyHandler comm.Handler
)

func SetAgencyHandler(handler comm.Handler) {
	agencyHandler = handler
}

// CurrentTr returns current Transport according the PL receiver.
func CurrentTr(addr *endp.Addr) txp.Trans {
	// We return CA's Transport which is transport between CA and its EA
	return ReceiverCA(addr).Trans()
}

// ReceiverCA returns the CA which decrypts PL.
func ReceiverCA(cnxAddr *endp.Addr) comm.Receiver {
	handlerKey := cnxAddr.PayloadTransportDID()
	return agent(handlerKey)
}

func AddHandler(endpoint Endpoint, handler ssi.Agent) {
	handlers.Lock()
	defer handlers.Unlock()
	handlers.m[endpoint] = handler.(comm.Handler)
}

func AddSeedHandler(endpoint Endpoint, handler comm.SeedHandler) {
	seedHandlers.Lock()
	defer seedHandlers.Unlock()
	seedHandlers.m[endpoint] = handler
}

func Handler(endpoint Endpoint) (handler comm.Handler) {
	if endp.IsInEndpoints(endpoint) {
		return nil
	}
	handlers.RLock()
	defer handlers.RUnlock()
	handler = handlers.m[endpoint]
	return handler
}

// IsHandlerInThisAgency checks prepares the endpoint. If the valid endpoint
// doesn't yet have a CA, it will be constructed in the handlerFromSeed()
func IsHandlerInThisAgency(endpoint *endp.Addr) (is bool) {
	if endp.IsInEndpoints(endpoint.PlRcvr) {
		return true
	}

	handlers.RLock()
	_, found := handlers.m[endpoint.PlRcvr]
	handlers.RUnlock()

	if !found {
		found = handlerFromSeed(endpoint)
	}
	return found
}

// handlerFromSeed try to find the seed handler for the endpoint and moves the
// seed to active handlers by constructing a CA for the endpoint if seed handler
// is available.
func handlerFromSeed(endp *endp.Addr) bool {
	seedHandlers.RLock()
	seed, ok := seedHandlers.m[endp.PlRcvr]
	seedHandlers.RUnlock()

	h, err := seed.Prepare()
	if !ok || err != nil {
		glog.V(3).Info("cannot find seed for endpoint: ",
			endp.PlRcvr)
		return false
	}
	seedHandlers.Lock()
	delete(seedHandlers.m, endp.PlRcvr)
	seedHandlers.Unlock()

	handlers.Lock()
	defer handlers.Unlock()
	handlers.m[endp.PlRcvr] = h
	return true
}

// RcvrCA returns the CA which is the actual PL receiver and Handler.
func RcvrCA(cnxAddr *endp.Addr) comm.Receiver {
	handlerKey := cnxAddr.ReceiverDID()
	return agent(handlerKey)
}

// agent returns CA by DID.
func agent(did string) comm.Receiver {
	handlerKey := did
	receivingHandler := Handler(handlerKey)
	agent := receivingHandler.(comm.Receiver)
	return agent
}

// Server calls this when it receives HTTP request.
func APICall(endpointAddress *endp.Addr,
	received didcomm.Payload) (response didcomm.Payload) {

	handlerKey := endpointAddress.ReceiverDID()
	receivingHandler := Handler(handlerKey)

	if receivingHandler == nil {
		receivingHandler = agencyHandler
	}
	response, _ = receivingHandler.InOutPL(endpointAddress, received)
	return response
}

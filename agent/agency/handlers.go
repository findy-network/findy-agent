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
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/txp"
	"github.com/golang/glog"
)

const (
	ProtocolPath = "a2a" // default for A2A Protocols (Aries), serviceName2
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

func SeedHandlerCount() int {
	seedHandlers.RLock()
	defer seedHandlers.RUnlock()
	return len(seedHandlers.m)
}

func HandlerCount() int {
	handlers.RLock()
	defer handlers.RUnlock()
	return len(handlers.m)
}

// TODO LAPI: endpoint type and name of the argument is misleading
func Handler(endpoint Endpoint) (handler comm.Handler) {
	if endp.IsInEndpoints(endpoint) {
		return nil
	}
	handlers.RLock()
	defer handlers.RUnlock()
	handler = handlers.m[endpoint]
	return handler
}

// IsHandlerInThisAgency checks and prepares the endpoint. If the valid endpoint
// doesn't yet have a CA, it will be constructed from a seed. This is called
// intensively during the agency's run. It is somewhat optimized with read locks
// and lazy fetch.
func IsHandlerInThisAgency(rcvrDID string) (is bool) {
	if endp.IsInEndpoints(rcvrDID) {
		return true
	}

	// Checking must start from seeds. If an EA contacts same time from 2
	// different client, only other can build a handler. Still both must get
	// the handler. Other one builds it and other one gets it from the handlers
	// map.

	seedHandlers.RLock()
	_, isStillSeed := seedHandlers.m[rcvrDID]
	seedHandlers.RUnlock()
	if isStillSeed {
		buildHandlerFromSeed(rcvrDID)
	}

	handlers.RLock()
	_, found := handlers.m[rcvrDID]
	handlers.RUnlock()

	return found
}

// buildHandlerFromSeed gets the seed handler for the endpoint and moves the
// seed to active handlers by constructing a CA for the endpoint if seed handler
// is available.
func buildHandlerFromSeed(rcvrDID string) {
	seedHandlers.Lock()
	defer seedHandlers.Unlock()

	// Try to get seed with the write lock on. If cannot get it it means other
	// instance of us got it already and it will build the handler.
	seed, ok := seedHandlers.m[rcvrDID]
	if !ok {
		glog.V(3).Info("cannot find seed for endpoint anymore: ",
			rcvrDID)
		return
	}

	h, err := seed.Prepare()
	if err != nil {
		glog.V(3).Info("cannot build a handler from a seed: ",
			rcvrDID)
		return
	}
	delete(seedHandlers.m, rcvrDID)

	handlers.Lock()
	defer handlers.Unlock()
	handlers.m[rcvrDID] = h
	return
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

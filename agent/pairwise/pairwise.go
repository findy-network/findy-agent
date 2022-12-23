package pairwise

import (
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/core"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

type Pairwise struct {
	agent       ssi.Agent // agent which is the controller of this pairwise: caller or callee
	RoutingKeys []string
	DID         string
	VerKey      string
	Endp        string // name of the endpoint
	Name        string // name of the pairwise, used when stored to wallet
}

type Callee struct {
	Pairwise
	Caller core.DID
	Callee core.DID
}

// MARK: Callee ---

func (p *Callee) startStore() {
	p.Caller.Store(p.agent.ManagedWallet())

	_, storageH := p.agent.ManagedWallet()
	p.Callee.SavePairwiseForDID(storageH, p.Caller, core.PairwiseMeta{
		Name:  p.Name,
		Route: p.RoutingKeys,
	})
}

func (p *Callee) storeResult() error {
	return p.Caller.StoreResult()
}

func NewCalleePairwise(
	agent ssi.Agent,
	routingKeys []string,
	caller core.DID,
	pwName string,
	endp service.Addr,
) (p *Callee) {

	return &Callee{
		Caller: caller,
		Pairwise: Pairwise{
			agent:       agent,
			RoutingKeys: routingKeys,
			Endp:        endp.Endp,
			Name:        pwName,
		},
	}
}

func (p *Callee) CheckPreallocation(cnxAddr *endp.Addr) {
	defer err2.Catch(func(err error) {
		glog.Errorf("Error loading connection: %s (%v)", cnxAddr.ConnID, err)
	})

	// ssi.DIDAgent implements comm.Receiver interface
	// ssi.Agent is other interface, that's why the cast
	a := p.agent.(comm.Receiver)

	conn := try.To1(a.FindPWByID(cnxAddr.ConnID))
	p.Callee = a.LoadDID(conn.MyDID)

	assert.That(p.Callee != nil, "now we relay working pre-alloc")
	glog.V(3).Infoln("Responder/callee DID:", p.Callee.URI())
}

func (p *Callee) Store() (err error) {
	defer err2.Handle(&err)

	p.startStore() // Save their DID and pairwise info
	// Check the result for error handling AND for consuming async's result
	try.To(p.storeResult())

	return nil
}

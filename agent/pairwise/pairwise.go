package pairwise

import (
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
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

func (p *Callee) ConnReqToRespWithSet(
	f func(m didcomm.PwMsg),
) (
	respMsg didcomm.PwMsg,
	err error,
) {
	defer err2.Return(&err)

	/*reqDoc := p.Msg.FieldObj().(*didexchange.Request).Connection.DIDDoc
	assert.That(reqDoc != nil)

	responseMsg := p.respMsgAndOurDID()
	p.Name = p.Msg.Nonce()

	connReqDID := p.Msg.Did()

	var callerDID core.DID
	if method.DIDType(connReqDID) == method.TypePeer {
		docBytes := try.To1(json.Marshal(reqDoc))
		callerDID = try.To1(p.agent.NewOutDID(connReqDID, string(docBytes)))
	} else { // did:sov: is the default still
		// old 160-connection protocol handles old DIDs as plain
		rawDID := strings.TrimPrefix(connReqDID, "did:sov:")
		if rawDID == connReqDID {
			connReqDID = "did:sov:" + rawDID
			glog.V(3).Infoln("+++ normalizing Did()",
				rawDID, " ==>", connReqDID)
		}

		connReqVK := p.Msg.VerKey()
		callerDID = try.To1(p.agent.NewOutDID(connReqDID, connReqVK))
		p.agent.AddDIDCache(callerDID.(*ssi.DID))
	}

	f(responseMsg) // let caller set msg values

	p.Caller = callerDID // this MUST be before next line!*/
	p.startStore() // Save their DID and pairwise info

	// responseMsg := p.respMsgAndOurDID()
	// respMsg = responseMsg

	// Check the result for error handling AND for consuming async's result
	try.To(p.storeResult())

	return nil, nil
}

/*
func (p *Callee) respMsgAndOurDID() (msg didcomm.PwMsg) {
	if p.Callee == nil {
		glog.Warning("------ no enough information to create DID ------")
		p.Callee = try.To1(p.agent.NewDID(utils.Settings.DIDMethod(), ""))
	}
	responseMsg := p.factor.Create(didcomm.MsgInit{
		DIDObj:   p.Callee,
		Nonce:    p.Name,
		Name:     p.Msg.Nonce(),
		Endpoint: p.Msg.Endpoint().Endp,
	}).(didcomm.PwMsg)
	return responseMsg
}
*/

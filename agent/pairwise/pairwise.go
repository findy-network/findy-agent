package pairwise

import (
	"encoding/json"

	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-agent/method"
	"github.com/findy-network/findy-agent/std/didexchange"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

type Pairwise struct {
	agent    ssi.Agent     // agent which is the controller of this pairwise: caller or callee
	Msg      didcomm.PwMsg // payload's inner message which will build by multiple functions
	Endp     string        // name of the endpoint
	Name     string        // name of the pairwise, used when stored to wallet
	connType string        // ConnOffer / ConnHandshake or Pairwise / Handshake
	factor   didcomm.MsgFactor
}

type Callee struct {
	Pairwise
	Caller core.DID
	Callee core.DID
}

// MARK: Callee ---

func (p *Callee) startStore() {
	p.Caller.Store(p.agent.ManagedWallet())
	pwName := p.pairwiseName()

	// Find the routing keys from the request
	route := []string{}
	if req, ok := p.Msg.FieldObj().(*didexchange.Request); ok {
		route = didexchange.RouteForConnection(req.Connection)
	} else {
		glog.Warning("Callee.startStore() - no DIDExchange request found")
	}
	_, storageH := p.agent.ManagedWallet()
	p.Callee.SavePairwiseForDID(storageH, p.Caller, core.PairwiseMeta{
		Name:  pwName,
		Route: route,
	})
}

func (p *Callee) storeResult() error {
	return p.Caller.StoreResult()
}

func NewCalleePairwise(msgFactor didcomm.MsgFactor, agent ssi.Agent,
	msg didcomm.PwMsg) (p *Callee) {

	return &Callee{
		Pairwise: Pairwise{
			agent:  agent,
			Msg:    msg,
			Endp:   msg.Endpoint().Endp,
			Name:   msg.Nonce(),
			factor: msgFactor,
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

	reqDoc := p.Msg.FieldObj().(*didexchange.Request).Connection.DIDDoc
	assert.That(reqDoc != nil)

	responseMsg := p.respMsgAndOurDID()
	p.Name = p.Msg.Nonce()

	connReqDID := p.Msg.Did()

	var callerDID core.DID
	if method.DIDType(connReqDID) == method.TypePeer {
		docBytes := try.To1(json.Marshal(reqDoc))
		callerDID = try.To1(p.agent.NewOutDID(connReqDID, string(docBytes)))
	} else { // did:sov: is the default still
		connReqVK := p.Msg.VerKey()
		callerDID = try.To1(p.agent.NewOutDID(connReqDID, connReqVK))
		p.agent.AddDIDCache(callerDID.(*ssi.DID))
	}

	f(responseMsg) // let caller set msg values

	p.Caller = callerDID // this MUST be before next line!
	p.startStore()       // Save their DID and pairwise info

	respMsg = responseMsg

	// Check the result for error handling AND for consuming async's result
	try.To(p.storeResult())

	return respMsg, nil
}

func (p *Callee) respMsgAndOurDID() (msg didcomm.PwMsg) {
	if p.Callee == nil {
		glog.Warning("------ no enough information to create DID ------")
		p.Callee = try.To1(p.agent.NewDID(utils.Settings.DIDMethod(), ""))
	}
	responseMsg := p.factor.Create(didcomm.MsgInit{
		DIDObj:   p.Callee,
		Nonce:    p.Msg.Nonce(),
		Name:     p.Msg.Nonce(),
		Endpoint: p.Msg.Endpoint().Endp,
	}).(didcomm.PwMsg)
	return responseMsg
}

// MARK: Pairwise methods

func (p *Pairwise) pairwiseName() string {
	switch {
	case p.connType == pltype.ConnectionTrustAgent:
		return pltype.ConnectionTrustAgent
	case p.connType == pltype.ConnectionHandshake && p.agent.IsCA():
		return pltype.HandshakePairwiseName
	default:
		return p.Name
	}
}

package pairwise

import (
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/std/didexchange"
	"github.com/findy-network/findy-wrapper-go/did"
	"github.com/golang/glog"
	"github.com/lainio/err2"
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
	Caller *ssi.DID
	Callee *ssi.DID
}

// MARK: Callee ---

func (p *Callee) startStore() {
	wallet := p.agent.Wallet()
	p.Caller.Store(wallet)
	pwName := p.pairwiseName()

	// Find the routing keys from the request
	route := []string{}
	if req, ok := p.Msg.FieldObj().(*didexchange.Request); ok {
		route = didexchange.RouteForConnection(req.Connection)
	} else {
		glog.Warning("Callee.startStore() - no DIDExchange request found")
	}

	p.Callee.SavePairwiseForDID(wallet, p.Caller, ssi.PairwiseMeta{
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
	a := p.agent.(comm.Receiver)
	calleeDID := a.LoadDID(cnxAddr.RcvrDID)
	r := <-did.Meta(a.Wallet(), calleeDID.Did())
	err2.Check(r.Err())
	if r.Str1() == cnxAddr.EdgeToken {
		glog.V(1).Infoln("==== using preallocated pw DID ====", calleeDID.Did())
		p.Callee = calleeDID
	} else {
		glog.V(1).Infoln("===== Cannot use pw DID, NO META =====")
	}
}

func (p *Callee) ConnReqToRespWithSet(
	f func(m didcomm.PwMsg),
) (
	respMsg didcomm.PwMsg,
	err error,
) {
	defer err2.Return(&err)

	responseMsg := p.respMsgAndOurDID()
	p.Name = p.Msg.Nonce()
	connReqDID := p.Msg.Did()
	connReqVK := p.Msg.VerKey()
	callerDID := ssi.NewDid(connReqDID, connReqVK)
	p.agent.AddDIDCache(callerDID)

	f(responseMsg) // let caller set msg values

	p.Caller = callerDID // this MUST be before next line!
	p.startStore()       // Save their DID and pairwise info

	respMsg = responseMsg

	// Check the result for error handling AND for consuming async's result
	err2.Check(p.storeResult())

	return respMsg, nil
}

func (p *Callee) respMsgAndOurDID() (msg didcomm.PwMsg) {
	if p.Callee == nil {
		p.Callee = p.agent.CreateDID("")
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

package pairwise

import (
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-wrapper-go"
	"github.com/findy-network/findy-wrapper-go/did"
)

type Saver interface {
	StartStore()
	StoreResult() error
	SaveEndpoint(addr string)
	MeDID() string
	YouDID() string
}

type Pairwise struct {
	agent    ssi.Agent     // agent which is the controller of this pairwise: caller or callee
	Msg      didcomm.PwMsg // payload's inner message which will build by multiple functions
	Endp     string        // name of the endpoint
	Name     string        // name of the pairwise, used when stored to wallet
	connType string        // ConnOffer / ConnHandshake or Pairwise / Handshake
	factor   didcomm.MsgFactor
}

type Caller struct {
	Pairwise
	caller     *ssi.DID
	callerRoot *ssi.DID
	callee     *ssi.DID
}

func (p *Caller) MeDID() string {
	if p.callee == nil || p.caller == nil {
		return ""
	}
	return p.caller.Did()
}

func (p *Caller) YouDID() string {
	if p.callee == nil || p.caller == nil {
		return ""
	}
	return p.callee.Did()
}

func (p *Caller) SaveEndpoint(addr string) {
	theirDID := p.callee
	p.saveEndpoint(theirDID.Did(), addr, theirDID.VerKey())
}

type Callee struct {
	Pairwise
	Caller *ssi.DID
	Callee *ssi.DID
}

func (p *Callee) MeDID() string {
	if p.Callee == nil || p.Caller == nil {
		return ""
	}
	return p.Callee.Did()
}

func (p *Callee) YouDID() string {
	if p.Callee == nil || p.Caller == nil {
		return ""
	}
	return p.Caller.Did()
}

func (p *Callee) SaveEndpoint(addr string) {
	theirDID := p.Caller
	p.saveEndpoint(theirDID.Did(), addr, theirDID.VerKey())
}

func NewCallerPairwise(msgFactor didcomm.MsgFactor, callerAgent ssi.Agent,
	callerRootDid *ssi.DID, connType string) (p *Caller) {

	return &Caller{
		Pairwise: Pairwise{
			factor:   msgFactor,
			agent:    callerAgent,
			connType: connType,
		},
		callerRoot: callerRootDid,
	}
}

func (p *Caller) Build(attachToAgency bool) {
	if p.agent.IsEA() {
		p.caller = p.agent.CreateDID("")
		p.Msg = p.factor.Create(didcomm.MsgInit{
			Did:      p.caller.Did(),
			VerKey:   p.caller.VerKey(),
			Nonce:    utils.NewNonceStr(),
			Endpoint: p.Endp,
			Name:     p.Name,
		}).(didcomm.PwMsg)
	} else {
		p.caller = p.agent.CreateDID("")
		p.buildMsg(attachToAgency)

		if !utils.Settings.LocalTestMode() { // no ledger writes in test mode
			p.agent.SendNYM(p.caller, p.callerRoot.Did(), findy.NullString, findy.NullString)
		}
	}
}

func (p *Caller) buildMsg(attachToAgency bool) {
	var nonce uint64
	if attachToAgency { // if this Handshake / On-boarding we need this only then
		agency.AddHandler(p.caller.Did(), p.agent)
		nonce = utils.NewNonce()
		agency.Register.Add(p.agent.RootDid().Did(), p.Endp, p.caller.Did())
	}

	ns := utils.NonceToStr(nonce)
	p.Msg = p.factor.Create(didcomm.MsgInit{
		Did:      p.caller.Did(),
		VerKey:   p.caller.VerKey(),
		Nonce:    ns,
		Endpoint: p.Name,
		Name:     p.Name,
	}).(didcomm.PwMsg)
}

func (p *Caller) ReceiveResponse(encryptedResponse string) didcomm.PwMsg {
	decryptedMsg := p.factor.NewAnonDecryptedMsg(p.agent.Wallet(), encryptedResponse, p.caller)
	p.processMessage(decryptedMsg)
	return decryptedMsg
}

func (p *Caller) processMessage(decryptedMsg didcomm.PwMsg) {
	// get the DID from decrypted msg data values
	p.callee = ssi.NewDid(decryptedMsg.Did(), decryptedMsg.VerKey())

	// This pairwise is now ready to be saved to agent's wallet
	p.StartStore()

	// Only a cloud agent can write to the ledger where we don't write on test mode
	if p.agent.IsCA() && !utils.Settings.LocalTestMode() {
		p.agent.SendNYM(p.callee, p.callerRoot.Did(), findy.NullString, findy.NullString)
	}

	// Check the result for error handling AND for consuming async's result
	if err := p.StoreResult(); err != nil {
		glog.Error("error in finalizing a pairwise: ", err)
	}
}

func (p *Caller) StartStore() {
	wallet := p.agent.Wallet()
	p.callee.Store(wallet)
	pwName := p.pairwiseName()
	p.caller.Pairwise(wallet, p.callee, pwName)
}

func (p *Caller) StoreResult() error {
	return p.callee.StoreResult()
}

func (p *Caller) Bind(msg didcomm.PwMsg) {
	p.processMessage(msg)
}

// MARK: Callee ---

func (p *Callee) StartStore() {
	//log.Println("CalleePw StartStore()")
	wallet := p.agent.Wallet()
	p.Caller.Store(wallet)
	pwName := p.pairwiseName()
	p.Callee.Pairwise(wallet, p.Caller, pwName)
}

func (p *Callee) StoreResult() error {
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

func (p *Callee) ConnReqToResp() (respMsg didcomm.PwMsg) {
	responseMsg := p.RespMsgAndOurDID()
	p.Name = p.Msg.Name()
	connReqDID := p.Msg.Did()
	connReqVK := p.Msg.VerKey()
	callerDID := ssi.NewDid(connReqDID, connReqVK)
	p.agent.AddDIDCache(callerDID)
	p.Caller = callerDID // this MUST be before next line!
	builder, ok := p.agent.(ssi.EndpointBuilder)
	if !ok {
		glog.Error("programming error")
		panic("shouldn't be here!")
	}
	responseMsg.SetEndpoint(service.Addr{Endp: builder.BuildEndpURL()})
	p.StartStore() // Save their DID and pairwise info
	respMsg = responseMsg.(didcomm.Msg).AnonEncrypt(callerDID)

	// Check the result for error handling AND for consuming async's result
	if err := p.StoreResult(); err != nil {
		glog.Error("error in finalizing the pairwise: ", err)
	}
	return respMsg
}

func (p *Callee) ConnReqToRespWithSet(
	f func(m didcomm.PwMsg)) (respMsg didcomm.PwMsg, err error) {

	defer err2.Return(&err)

	responseMsg := p.RespMsgAndOurDID()
	p.Name = p.Msg.Nonce()
	connReqDID := p.Msg.Did()
	connReqVK := p.Msg.VerKey()
	callerDID := ssi.NewDid(connReqDID, connReqVK)
	p.agent.AddDIDCache(callerDID)

	f(responseMsg) // let caller set msg values

	p.Caller = callerDID // this MUST be before next line!
	p.StartStore()       // Save their DID and pairwise info

	respMsg = responseMsg

	// Check the result for error handling AND for consuming async's result
	err2.Check(p.StoreResult())

	return respMsg, nil
}

func (p *Callee) RespMsgAndOurDID() (msg didcomm.PwMsg) {
	p.Callee = p.agent.CreateDID("")
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

func (p *Pairwise) saveEndpoint(DID, addr, key string) {
	//log.Printf("saveEndp(%v, %v)\n", DID, addr)
	r := <-did.SetEndpoint(p.agent.Wallet(), DID, addr, key)
	if r.Err() != nil {
		panic(r.Error())
	}
}

package connection

import (
	"errors"

	"github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pairwise"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/sec"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/std/decorator"
	diddoc "github.com/findy-network/findy-agent/std/did"
	"github.com/findy-network/findy-agent/std/didexchange"
	"github.com/findy-network/findy-wrapper-go/did"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

type statusPairwise struct {
	Name          string `json:"name"`
	MyDID         string `json:"myDid"`
	TheirDID      string `json:"theirDid"`
	TheirEndpoint string `json:"theirEndpoint"`
	TheirLabel    string `json:"theirLabel"`
}

var connectionProcessor = comm.ProtProc{
	Starter: startConnectionProtocol,
	Handlers: map[string]comm.HandlerFunc{
		pltype.HandlerResponse: handleConnectionResponse,
		pltype.HandlerRequest:  handleConnectionRequest,
	},
	Status: getPairwiseStatus,
}

func init() {
	prot.AddStarter(pltype.CAPairwiseCreate, connectionProcessor)
	prot.AddStatusProvider(pltype.AriesProtocolConnection, connectionProcessor)
	comm.Proc.Add(pltype.AriesProtocolConnection, connectionProcessor)
}

func startConnectionProtocol(ca comm.Receiver, task *comm.Task) {
	defer err2.CatchTrace(func(err error) {
		glog.Error("ERROR in starting connection protocol:", err)
	})

	task.ReceiverEndp = service.Addr{
		Endp: task.ConnectionInvitation.ServiceEndpoint,
		Key:  task.ConnectionInvitation.RecipientKeys[0],
	}

	meAddr := ca.CAEndp(true) // CA can give us w-EA's endpoint
	me := ca.WDID()
	wa := ca.WorkerEA()
	ssiWA := wa.(ssi.Agent)

	caller := ssiWA.CreateDID("")  // Create a new DID for our end
	pubEndp := *meAddr             // and build an endpoint for..
	pubEndp.RcvrDID = caller.Did() // our new PW DID

	// build a connection message to send to another agent
	msg := didexchange.NewRequest(&didexchange.Request{
		Label: task.Info,
		Connection: &didexchange.Connection{
			DID:    caller.Did(),
			DIDDoc: diddoc.NewDoc(caller, pubEndp.AE()),
		},
		// when out-of-bound and did-exchange protocols are supported we
		// should start to save connection_id to Thread.PID
		Thread: &decorator.Thread{ID: task.ConnectionInvitation.ID},
	})
	// add to the cache until all lazy fetches are called
	ssiWA.AddDIDCache(caller)

	// Write EA's new DID (caller) to CA's wallet (e.g. routing) and to ledger
	err2.Check(ca.SaveTheirDID(caller.Did(), caller.VerKey(), true))

	// Save needed data to PSM related Pairwise Representative
	pwr := &psm.PairwiseRep{
		Key:        psm.StateKey{DID: me, Nonce: task.Nonce},
		Name:       task.Nonce,
		TheirLabel: task.ConnectionInvitation.Label,
		Caller:     psm.DIDRep{DID: caller.Did(), VerKey: caller.VerKey(), My: true},
		Callee:     psm.DIDRep{},
	}
	err2.Check(psm.AddPairwiseRep(pwr))

	// Create payload to send
	opl := aries.PayloadCreator.NewMsg(task.Nonce, pltype.AriesConnectionRequest, msg)

	// Create secure pipe to send payload to other end of the new PW
	secPipe := sec.NewPipeByVerkey(caller, task.ReceiverEndp.Key)
	wa.AddPipeToPWMap(*secPipe, pwr.Name)

	// Update PSM state, and send the payload to other end
	err2.Check(prot.UpdatePSM(me, caller.Did(), task, opl, psm.Sending))
	err2.Check(comm.SendPL(*secPipe, task, opl))

	// Sending went OK, update PSM once again
	wpl := mesg.NewPayloadBase(task.Nonce, pltype.AriesConnectionResponse)
	err2.Check(prot.UpdatePSM(me, caller.Did(), task, wpl, psm.Waiting))
}

func handleConnectionResponse(packet comm.Packet) (err error) {
	defer err2.Annotate("connection response", &err)

	meDID := packet.Receiver.Trans().MessagePipe().In.Did()
	ipl := packet.Payload
	cnxAddr := packet.Address
	a := packet.Receiver

	nonce := ipl.ThreadID()
	glog.V(1).Infoln("current thread ID:", nonce)
	if cnxAddr.EdgeToken != "" {
		glog.V(1).Infoln("****** using URL for nonce in RESPONSE", cnxAddr.EdgeToken)
		nonce = cnxAddr.EdgeToken
	}

	response := ipl.MsgHdr().FieldObj().(*didexchange.Response)

	if !err2.Bool.Try(response.Verify()) {
		glog.Error("cannot verify Connection Response signature --> send NACK")
		return errors.New("cannot verify connection response signature")
		// todo: send NACK here
	}

	task := comm.NewTaskFromConnectionResponse(ipl, response)

	err2.Check(prot.UpdatePSM(meDID, "", task, ipl, psm.Received))

	pwr, err := psm.GetPairwiseRep(psm.StateKey{DID: meDID, Nonce: nonce})
	err2.Check(err)
	msgMeDID := pwr.Caller.DID
	caller := a.LoadDID(pwr.Caller.DID)

	im := ipl.MsgHdr().(didcomm.PwMsg)

	// Set pairwise info about other end to wallet
	callee := ssi.NewDid(im.Did(), im.VerKey())
	callee.Store(a.Wallet())

	pwName := pwr.Name

	caller.Pairwise(a.Wallet(), callee, pwName)

	// SAVE ENDPOINT to wallet
	calleeEndp := endp.NewAddrFromPublic(im.Endpoint())
	r := <-did.SetEndpoint(a.Wallet(), callee.Did(), calleeEndp.Address(), calleeEndp.VerKey)
	err2.Check(r.Err())

	// Save Rep and PSM
	newPwr := &psm.PairwiseRep{
		Key:        pwr.Key,
		Name:       pwr.Name,
		TheirLabel: pwr.TheirLabel,
		Callee:     psm.DIDRep{DID: callee.Did(), VerKey: calleeEndp.VerKey, Endp: calleeEndp.Address(), My: false},
		Caller:     pwr.Caller,
	}
	err2.Check(psm.AddPairwiseRep(newPwr)) // updates the previously created

	// It's important to SAVE new pairwise's DIDs to our CA's wallet for
	// future routing. Everything goes thru CA.
	ca := agency.RcvrCA(cnxAddr)
	err2.Check(ca.SaveTheirDID(callee.Did(), callee.VerKey(), false))
	// Caller DID saved when we sent Conn_Req, in case both parties are us

	callee.SetAEndp(im.Endpoint())
	a.AddToPWMap(caller, callee, pwName) // to access PW later, map it

	// Update that PSM is successfully Ready
	emptyMsg := aries.MsgCreator.Create(didcomm.MsgInit{})
	opl := aries.PayloadCreator.NewMsg(utils.UUID(), pltype.AriesConnectionResponse, emptyMsg)
	err2.Check(prot.UpdatePSM(meDID, msgMeDID, task, opl, psm.ReadyACK))

	return nil
}

func handleConnectionRequest(packet comm.Packet) (err error) {
	defer err2.Annotate("connection req", &err)

	// The agent DID, the PW DID is msgMeDID below
	meDID := packet.Receiver.Trans().MessagePipe().In.Did()
	msgMeDID := "" // not known yet, will set it after pw is made

	ipl := packet.Payload
	cnxAddr := packet.Address
	a := packet.Receiver

	safeThreadID := ipl.ThreadID()
	connectionID := safeThreadID
	if cnxAddr.EdgeToken != "" {
		glog.V(1).Infoln("=== using URL edge, safe is", cnxAddr.EdgeToken, safeThreadID)
		connectionID = cnxAddr.EdgeToken
	}

	req := ipl.MsgHdr().FieldObj().(*didexchange.Request)
	task := comm.NewTaskFromRequest(ipl, req, safeThreadID)
	task.ReceiverEndp = cnxAddr.AE()

	err2.Check(prot.UpdatePSM(meDID, msgMeDID, task, ipl, psm.Received))

	task.SwitchDirection()

	// MARK: we must switch the Nonce for pairwise construction. We will return
	//  it back after we are done. This is because AcaPy compatibility
	ipl.MsgHdr().Thread().ID = connectionID

	calleePw := pairwise.NewCalleePairwise(
		didexchange.ResponseCreator, a.(ssi.Agent), ipl.MsgHdr().(didcomm.PwMsg))

	msg, err := calleePw.ConnReqToRespWithSet(func(m didcomm.PwMsg) {
		msgMeDID = m.Did() // set our pw DID

		// calculate our endpoint for the pairwise
		pubEndp := *cnxAddr         // set our agent's URL as a base addr
		pubEndp.RcvrDID = m.Did()   // set our pw DID to actual agent DID in addr
		pubEndp.VerKey = m.VerKey() // set our pw VerKey as well

		m.SetEndpoint(service.Addr{
			Endp: pubEndp.Address(),
			Key:  pubEndp.VerKey,
		})
	})
	err2.Check(err)

	// MARK: we must switch the Nonce for pairwise construction back. NOW we
	//  return it back after we are done. This is because AcaPy compatibility
	ipl.MsgHdr().Thread().ID = safeThreadID
	// MARK: very very important to rollback this as well
	glog.V(1).Infoln("=== msg.Thread.ID", msg.Thread().ID, safeThreadID)
	msg.Thread().ID = safeThreadID

	IncomingPWMsg := ipl.MsgHdr().(didcomm.PwMsg) // incoming pairwise message
	caller := calleePw.Caller                     // the other end, we'r here the callee
	callerEndp := endp.NewAddrFromPublic(IncomingPWMsg.Endpoint())
	callerAddress := callerEndp.Address()
	pwr := &psm.PairwiseRep{
		Key:        psm.StateKey{DID: meDID, Nonce: connectionID},
		Name:       connectionID,
		TheirLabel: req.Label,
		Callee:     psm.DIDRep{DID: calleePw.Callee.Did(), VerKey: calleePw.Callee.VerKey(), My: true},
		Caller:     psm.DIDRep{DID: caller.Did(), VerKey: caller.VerKey(), Endp: callerAddress},
	}
	err2.Check(psm.AddPairwiseRep(pwr))

	// SAVE ENDPOINT to wallet
	r := <-did.SetEndpoint(a.Wallet(), caller.Did(), callerAddress, callerEndp.VerKey)
	err2.Check(r.Err())

	// It's important to SAVE new pairwise's DIDs to our CA's wallet for
	// future routing. Everything goes thru CA. NOTE, only those DIDs
	// which are created by us are written to the Ledger
	ca := agency.RcvrCA(cnxAddr)
	err2.Check(ca.SaveTheirDID(caller.Did(), caller.VerKey(), false))
	err2.Check(ca.SaveTheirDID(calleePw.Callee.Did(), calleePw.Callee.VerKey(), true))

	res := msg.FieldObj().(*didexchange.Response)
	pipe := sec.Pipe{
		In:  calleePw.Callee, // This is us
		Out: caller,          // This is the other end, who sent the Request
	}

	err2.Check(res.Sign(pipe)) // we must sign the Response before send it

	caller.SetAEndp(IncomingPWMsg.Endpoint())
	a.AddToPWMap(calleePw.Callee, caller, connectionID) // to access PW later, map it

	// build the response payload, update PSM, and send the PL with sec.Pipe
	opl := aries.PayloadCreator.NewMsg(utils.UUID(), pltype.AriesConnectionResponse, msg)
	err2.Check(prot.UpdatePSM(meDID, msgMeDID, task, opl, psm.Sending))
	err2.Check(comm.SendPL(pipe, task, opl))

	// update the PSM, we are ready at this end for this protocol
	emptyMsg := aries.MsgCreator.Create(didcomm.MsgInit{})
	wpl := aries.PayloadCreator.NewMsg(task.Nonce, pltype.AriesConnectionResponse, emptyMsg)
	err2.Check(prot.UpdatePSM(meDID, msgMeDID, task, wpl, psm.ReadyACK))

	return nil
}

func getPairwiseStatus(workerDID string, taskID string) interface{} {
	defer err2.CatchTrace(func(err error) {
		glog.Error("Failed to set connection status: ", err)
	})

	key := &psm.StateKey{
		DID:   workerDID,
		Nonce: taskID,
	}

	pw, err := psm.GetPairwiseRep(*key)
	err2.Check(err)

	myDID := pw.Callee
	theirDID := pw.Caller
	theirEndpoint := pw.Caller.Endp

	if !myDID.My {
		myDID = pw.Caller
		theirDID = pw.Callee
		theirEndpoint = pw.Callee.Endp
	}

	return statusPairwise{
		Name:          pw.Name,
		MyDID:         myDID.DID,
		TheirDID:      theirDID.DID,
		TheirEndpoint: theirEndpoint,
		TheirLabel:    pw.TheirLabel,
	}
}

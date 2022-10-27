package connection

import (
	"encoding/gob"
	"encoding/json"
	"strings"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/managed"
	"github.com/findy-network/findy-agent/agent/pairwise"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/sec"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/ssi"
	storage "github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-agent/method"
	"github.com/findy-network/findy-agent/std/didexchange"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/findy-network/findy-common-go/std/didexchange/invitation"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"

	// import didexchange protocol structures to initialize message creators
	_ "github.com/findy-network/findy-agent/std/didexchange/v0"
	_ "github.com/findy-network/findy-agent/std/didexchange/v1"
)

type taskDIDExchange struct {
	comm.TaskBase
	Invitation invitation.Invitation
	Label      string
}

var connectionProcessor = comm.ProtProc{
	Creator: createConnectionTask,
	Starter: startConnectionProtocol,
	Handlers: map[string]comm.HandlerFunc{
		pltype.HandlerResponse: handleConnectionResponse, // to Caller (sends the request)
		pltype.HandlerRequest:  handleConnectionRequest,  // to Callee
		// TODO: add complete handler
	},
	FillStatus: fillPairwiseStatus,
}

func init() {
	gob.Register(&taskDIDExchange{})
	// handle both protocol formats - with and without s
	prot.AddCreator(pltype.ProtocolConnection, connectionProcessor)
	prot.AddCreator(pltype.AriesProtocolConnection, connectionProcessor)
	prot.AddCreator(pltype.AriesProtocolDIDExchange, connectionProcessor)
	prot.AddStarter(pltype.CAPairwiseCreate, connectionProcessor)
	prot.AddStarter(pltype.CAPairwiseInvitation, connectionProcessor)
	prot.AddStatusProvider(pltype.AriesProtocolConnection, connectionProcessor)
	prot.AddStatusProvider(pltype.AriesProtocolDIDExchange, connectionProcessor)
	comm.Proc.Add(pltype.AriesProtocolConnection, connectionProcessor)
	comm.Proc.Add(pltype.AriesProtocolDIDExchange, connectionProcessor)
}

func createConnectionTask(
	header *comm.TaskHeader,
	protocol *pb.Protocol,
) (
	t comm.Task,
	err error,
) {
	defer err2.Returnf(&err, "createConnectionTask")

	var inv invitation.Invitation
	var label string
	if protocol != nil {
		assert.That(
			protocol.GetDIDExchange() != nil,
			"didExchange protocol data missing")

		// Let's let invitation package translate incoming invitation. It will
		// handle two different type formats even the field name ends with
		// JSON.
		inv = try.To1(invitation.Translate(protocol.GetDIDExchange().GetInvitationJSON()))

		header.TaskID = inv.ID()
		label = protocol.GetDIDExchange().GetLabel()

		glog.V(1).Infof("Create task for DIDExchange with invitation id %s", inv.ID())
	}

	return &taskDIDExchange{
		TaskBase:   comm.TaskBase{TaskHeader: *header},
		Invitation: inv,
		Label:      label,
	}, nil
}

func startConnectionProtocol(ca comm.Receiver, task comm.Task) {
	defer err2.CatchTrace(func(err error) {
		glog.Error("ERROR in starting connection protocol:", err)
	})

	deTask, ok := task.(*taskDIDExchange)
	assert.That(ok)

	invMsg := aries.MsgCreator.Create(didcomm.MsgInit{
		Type: deTask.Invitation.Type(),
		AID:  deTask.Invitation.ID()}).(didexchange.PwMsg)

	connectionID := deTask.Invitation.ID()
	meAddr := ca.CAEndp(deTask.ID()) // CA can give us w-EA's endpoint
	me := ca.WDID()
	wa := ca.WorkerEA()
	ssiWA := wa.(ssi.Agent)

	if task.Role() == pb.Protocol_ADDRESSEE {
		glog.V(3).Infof("it's us who waits connection (%v) to invitation", deTask.Invitation.ID())
		pl, state := invMsg.Wait()
		try.To(prot.UpdatePSM(me, connectionID, task, pl, state))
		return
	}

	deTask.SetReceiverEndp(service.Addr{
		Endp: deTask.Invitation.Services()[0].ServiceEndpoint,
		Key:  deTask.Invitation.Services()[0].RecipientKeysAsB58()[0],
	})

	didMethod := task.DIDMethod()
	caller := try.To1(ssiWA.NewDID(didMethod, meAddr.Address())) // Create a new DID for our end

	addToSovCacheIf(ssiWA, caller)

	// Save needed data to PSM related Pairwise Representative
	pwr := &pairwiseRep{
		StateKey:   psm.StateKey{DID: me, Nonce: deTask.ID()},
		Name:       deTask.ID(),
		TheirLabel: deTask.Invitation.Label(),
		Caller:     didRep{DID: caller.Did(), VerKey: caller.VerKey(), My: true},
		Callee:     didRep{},
	}
	try.To(psm.AddRep(pwr))

	// Create secure pipe to send payload to other end of the new PW
	receiverKey := task.ReceiverEndp().Key
	receiverKeys := buildRouting(task.ReceiverEndp().Endp, receiverKey,
		deTask.Invitation.Services()[0].RoutingKeysAsB58(), didMethod)
	callee := try.To1(wa.NewOutDID(receiverKeys...))
	secPipe := sec.Pipe{In: caller, Out: callee}
	wa.AddPipeToPWMap(secPipe, pwr.Name)

	// Create payload to send
	opl, state := try.To2(invMsg.Next(deTask.Label, caller))

	// Update PSM state, and send the payload to other end
	try.To(prot.UpdatePSM(me, connectionID, task, opl, state))
	try.To(comm.SendPL(secPipe, task, opl))

	// Sending went OK, update PSM once again
	reqMsg := opl.FieldObj().(didexchange.PwMsg)
	wpl, state := reqMsg.Wait()
	try.To(prot.UpdatePSM(me, connectionID, task, wpl, state))
}

func addToSovCacheIf(ssiWA ssi.Agent, caller core.DID) {
	d, ok := caller.(*ssi.DID)
	if ok {
		// add to the cache until all lazy fetches are called
		ssiWA.AddDIDCache(d)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func buildRouting(addr, rKey string, rKeys []string, didMethod method.Type) []string {
	switch didMethod {
	case method.TypePeer:
		retval := make([]string, 2, max(2, len(rKeys)+1))
		retval[0] = didMethod.DIDString()
		doc := try.To1(method.NewDoc(rKey, addr))
		docBytes := try.To1(json.Marshal(doc))
		retval[1] = string(docBytes)
		return append(retval, rKeys...)

	// Defaults still are method.TypeSov, method.TypeIndy:
	default:
		retval := make([]string, 2, max(2, len(rKeys)+1))
		retval[0] = didMethod.DIDString()
		retval[1] = rKey
		return append(retval, rKeys...)
	}
}

// handleConnectionRequest is handled by 'responder' aka callee.
// The party who receives conn_req.
func handleConnectionRequest(packet comm.Packet) (err error) {
	defer err2.Returnf(&err, "connection req")

	// The agent DID
	meDID := packet.Receiver.MyDID().Did()

	ipl := packet.Payload
	cnxAddr := packet.Address
	receiver := packet.Receiver

	safeThreadID := ipl.ThreadID()
	connectionID := cnxAddr.ConnID

	reqMsg := ipl.MsgHdr().(didexchange.PwMsg)

	callerEP := reqMsg.Endpoint()
	receiverEP := cnxAddr.AE()

	task := &comm.TaskBase{
		TaskHeader: comm.TaskHeader{
			TaskID:   ipl.ThreadID(),
			TypeID:   ipl.Type(),
			Receiver: receiverEP,
			Sender:   callerEP,
		},
	}

	try.To(prot.UpdatePSM(meDID, connectionID, task, ipl, psm.Received))

	task.SwitchDirection()

	wca := receiver.(ssi.Agent)
	var callerDID core.DID
	if method.DIDType(reqMsg.Did()) == method.TypePeer {
		callerDID = try.To1(wca.NewOutDID(reqMsg.Did(), string(try.To1(reqMsg.DIDDocument().MarshalJSON()))))
	} else { // did:sov: is the default still
		// old 160-connection protocol handles old DIDs as plain
		rawDID := strings.TrimPrefix(reqMsg.Did(), "did:sov:")
		if rawDID == reqMsg.Did() {
			glog.V(3).Infoln("+++ normalizing Did()", rawDID, " ==>", reqMsg.Did())
			rawDID = "did:sov:" + rawDID
		}

		connReqVK := reqMsg.VerKey()
		callerDID = try.To1(wca.NewOutDID(rawDID, connReqVK))
		wca.AddDIDCache(callerDID.(*ssi.DID))
	}

	calleePw := pairwise.NewCalleePairwise(
		wca, reqMsg.RoutingKeys(), callerDID, connectionID, receiverEP)
	calleePw.CheckPreallocation(cnxAddr) // load our DID

	myEndp := *cnxAddr                       // set our agent's URL as a base addr
	myEndp.ConnID = connectionID             // set our pw DID to actual agent DID in addr
	myEndp.VerKey = calleePw.Callee.VerKey() // set our pw VerKey as well
	calleePw.Callee.SetAEndp(service.Addr{
		Endp: myEndp.Address(),
		Key:  myEndp.VerKey,
	})

	try.To1(calleePw.ConnReqToRespWithSet(func(m didexchange.PwMsg) {
		// Legacy: calculate our endpoint for the pairwise
		// when DIDDoc is used for all the DIDs extra info.

		// pubEndp := *cnxAddr           // set our agent's URL as a base addr
		// pubEndp.ConnID = connectionID // set our pw DID to actual agent DID in addr
		// pubEndp.VerKey = m.VerKey()   // set our pw VerKey as well

		// m.SetEndpoint(service.Addr{
		// 	Endp: pubEndp.Address(),
		// 	Key:  pubEndp.VerKey,
		// })
	}))

	// todo: send NACK here if fails
	// NOTE: verify can be done only after their DID is stored to KMS
	try.To(reqMsg.Verify(
		callerDID.Packager().Crypto(),
		callerDID.Packager().KMS(),
	))

	caller := calleePw.Caller // the other end, we're here the callee
	callerEndp := endp.NewAddrFromPublic(reqMsg.Endpoint())
	callerAddress := callerEndp.Address()
	pwr := &pairwiseRep{
		StateKey:   psm.StateKey{DID: meDID, Nonce: safeThreadID}, // check if this really must be connection id
		Name:       connectionID,
		TheirLabel: reqMsg.Label(),
		Callee:     didRep{DID: calleePw.Callee.Did(), VerKey: calleePw.Callee.VerKey(), My: true},
		Caller:     didRep{DID: caller.Did(), VerKey: caller.VerKey(), Endp: callerAddress},
	}
	try.To(psm.AddRep(pwr))

	// SAVE ENDPOINT to wallet
	try.To(saveConnectionEndpoint(managedStorage(receiver), connectionID, callerAddress))

	pipe := sec.Pipe{
		In:  calleePw.Callee, // This is us
		Out: caller,          // This is the other end, who sent the Request
	}

	caller.SetAEndp(callerEP)
	receiver.AddToPWMap(calleePw.Callee, caller, connectionID) // to access PW later, map it

	// build the response payload, update PSM, and send the PL with sec.Pipe
	opl, state := try.To2(reqMsg.Next("", calleePw.Callee))
	try.To(prot.UpdatePSM(meDID, connectionID, task, opl, state))
	try.To(comm.SendPL(pipe, task, opl))

	// update the PSM
	respMsg := opl.FieldObj().(didexchange.PwMsg)
	wpl, state := respMsg.Wait()
	try.To(prot.UpdatePSM(meDID, connectionID, task, wpl, state))

	return nil
}

func handleConnectionResponse(packet comm.Packet) (err error) {
	defer err2.Returnf(&err, "connection response")

	connectionID := packet.Address.ConnID
	meDID := packet.Receiver.MyDID().Did()
	ipl := packet.Payload
	receiver := packet.Receiver
	respMsg := ipl.MsgHdr().(didexchange.PwMsg)

	nonce := ipl.ThreadID()

	respEndp := respMsg.Endpoint()

	task := &comm.TaskBase{
		TaskHeader: comm.TaskHeader{
			TaskID:   ipl.ThreadID(),
			TypeID:   ipl.Type(),
			Receiver: respEndp,
			Sender:   respEndp,
		},
	}

	try.To(prot.UpdatePSM(meDID, connectionID, task, ipl, psm.Received))

	pwr := try.To1(getPairwiseRep(psm.StateKey{DID: meDID, Nonce: nonce}))
	msgMeDID := pwr.Caller.DID
	caller := receiver.LoadDID(msgMeDID)

	// Set pairwise info about other end to wallet
	var callee core.DID
	if method.TypePeer == utils.Settings.DIDMethod() {
		callee = receiver.LoadDID(respMsg.Did())
	} else { // default method is did:sov:
		callee = ssi.NewDid(respMsg.Did(), respMsg.VerKey())
	}

	callee.Store(receiver.ManagedWallet())

	// todo: send NACK here if fails
	// NOTE: verify can be done only after their DID is stored to KMS
	try.To(respMsg.Verify(
		callee.Packager().Crypto(),
		callee.Packager().KMS(),
	))

	pwName := pwr.Name
	route := respMsg.RoutingKeys()
	caller.SavePairwiseForDID(managedStorage(receiver), callee, core.PairwiseMeta{
		Name:  pwName,
		Route: route,
	})

	// SAVE ENDPOINT to wallet
	calleeEndp := endp.NewAddrFromPublic(respEndp)
	try.To(saveConnectionEndpoint(managedStorage(receiver), pwName, calleeEndp.Address()))

	// Save Rep and PSM
	newPwr := &pairwiseRep{
		StateKey:   pwr.StateKey,
		Name:       pwr.Name,
		TheirLabel: pwr.TheirLabel,
		Callee:     didRep{DID: callee.Did(), VerKey: calleeEndp.VerKey, Endp: calleeEndp.Address(), My: false},
		Caller:     pwr.Caller,
	}
	try.To(psm.AddRep(newPwr)) // updates the previously created

	callee.SetAEndp(respEndp)
	receiver.AddToPWMap(caller, callee, pwName) // to access PW later, map it

	opl, state := try.To2(respMsg.Next("", nil))
	// Update that PSM is successfully Ready
	if state.IsReady() {
		try.To(prot.UpdatePSM(meDID, connectionID, task, opl, state))
	} else {
		pipe := sec.Pipe{
			In:  caller,
			Out: callee,
		}

		try.To(prot.UpdatePSM(meDID, connectionID, task, opl, state))
		try.To(comm.SendPL(pipe, task, opl))

		// Sending went OK, update PSM once again
		completeMsg := opl.FieldObj().(didexchange.PwMsg)
		wpl, state := completeMsg.Wait()
		try.To(prot.UpdatePSM(meDID, connectionID, task, wpl, state))
	}

	return nil
}

func saveConnectionEndpoint(mgdStorage managed.Wallet, connectionID, theirEndpoint string) error {
	store := mgdStorage.Storage().ConnectionStorage()
	connection, _ := store.GetConnection(connectionID)
	if connection == nil {
		connection = &storage.Connection{
			ID: connectionID,
		}
	}
	connection.TheirEndpoint = theirEndpoint
	return store.SaveConnection(*connection)
}

func fillPairwiseStatus(workerDID string, taskID string, ps *pb.ProtocolStatus) *pb.ProtocolStatus {
	defer err2.CatchTrace(func(err error) {
		glog.Error("Failed to get connection status: ", err)
	})

	assert.D.True(ps != nil)

	key := psm.StateKey{
		DID:   workerDID,
		Nonce: taskID,
	}
	glog.V(4).Infoln("status for:", key)

	status := ps

	pw := try.To1(getPairwiseRep(key))

	myDID := pw.Callee
	theirDID := pw.Caller
	theirEndpoint := pw.Caller.Endp

	if !myDID.My {
		myDID = pw.Caller
		theirDID = pw.Callee
		theirEndpoint = pw.Callee.Endp
	}

	status.Status = &pb.ProtocolStatus_DIDExchange{DIDExchange: &pb.ProtocolStatus_DIDExchangeStatus{
		ID:            pw.Name,
		MyDID:         myDID.DID,
		TheirDID:      theirDID.DID,
		TheirEndpoint: theirEndpoint,
		TheirLabel:    pw.TheirLabel,
	}}

	return status
}

func managedStorage(a comm.Receiver) managed.Wallet {
	_, ms := a.ManagedWallet()
	return ms
}

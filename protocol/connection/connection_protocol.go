package connection

import (
	"encoding/gob"
	"encoding/json"
	"errors"

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
	"github.com/findy-network/findy-agent/std/common"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-agent/std/didexchange"
	"github.com/findy-network/findy-agent/std/didexchange/signature"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/findy-network/findy-common-go/std/didexchange/invitation"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

type taskDIDExchange struct {
	comm.TaskBase
	Invitation   invitation.Invitation
	InvitationID string
	Label        string
}

var connectionProcessor = comm.ProtProc{
	Creator: createConnectionTask,
	Starter: startConnectionProtocol,
	Handlers: map[string]comm.HandlerFunc{
		pltype.HandlerResponse: handleConnectionResponse, // to Caller (sends the request)
		pltype.HandlerRequest:  handleConnectionRequest,  // to Callee
	},
	FillStatus: fillPairwiseStatus,
}

func init() {
	gob.Register(&taskDIDExchange{})
	// handle both protocol formats - with and without s
	prot.AddCreator(pltype.ProtocolConnection, connectionProcessor)
	prot.AddCreator(pltype.AriesProtocolConnection, connectionProcessor)
	prot.AddStarter(pltype.CAPairwiseCreate, connectionProcessor)
	prot.AddStarter(pltype.CAPairwiseInvitation, connectionProcessor)
	prot.AddStatusProvider(pltype.AriesProtocolConnection, connectionProcessor)
	comm.Proc.Add(pltype.AriesProtocolConnection, connectionProcessor)
}

func createConnectionTask(
	header *comm.TaskHeader,
	protocol *pb.Protocol,
) (
	t comm.Task,
	err error,
) {
	defer err2.Annotate("createConnectionTask", &err)

	var inv invitation.Invitation
	var label string
	if protocol != nil {
		assert.P.True(
			protocol.GetDIDExchange() != nil,
			"didExchange protocol data missing")

		// Let's let invitation package translate incoming invitation. It will
		// handle two different type formats even the field name ends with
		// JSON.
		inv = try.To1(invitation.Translate(protocol.GetDIDExchange().GetInvitationJSON()))

		header.TaskID = inv.ID
		label = protocol.GetDIDExchange().GetLabel()

		glog.V(1).Infof("Create task for DIDExchange with invitation id %s", inv.ID)
	}

	return &taskDIDExchange{
		TaskBase:     comm.TaskBase{TaskHeader: *header},
		Invitation:   inv,
		InvitationID: inv.ID,
		Label:        label,
	}, nil
}

func startConnectionProtocol(ca comm.Receiver, task comm.Task) {
	defer err2.CatchTrace(func(err error) {
		glog.Error("ERROR in starting connection protocol:", err)
	})

	deTask, ok := task.(*taskDIDExchange)
	assert.P.True(ok)

	connectionID := deTask.InvitationID
	meAddr := ca.CAEndp(deTask.ID()) // CA can give us w-EA's endpoint
	me := ca.WDID()
	wa := ca.WorkerEA()
	ssiWA := wa.(ssi.Agent)

	if task.Role() == pb.Protocol_ADDRESSEE {
		glog.V(3).Infof("it's us who waits connection (%v) to invitation",
			deTask.InvitationID)

		wpl := aries.PayloadCreator.New(
			didcomm.PayloadInit{
				ID:   task.ID(),
				Type: pltype.AriesConnectionRequest,
			})
		try.To(prot.UpdatePSM(me, connectionID, task, wpl, psm.Waiting))
		return
	}

	deTask.SetReceiverEndp(service.Addr{
		Endp: deTask.Invitation.ServiceEndpoint,
		Key:  deTask.Invitation.RecipientKeys[0],
	})

	didMethod := task.DIDMethod()
	caller := try.To1(ssiWA.NewDID(didMethod, meAddr.Address())) // Create a new DID for our end

	// build a connection request message to send to another agent
	msg := didexchange.NewRequest(&didexchange.Request{
		Label: deTask.Label,
		Connection: &didexchange.Connection{
			DID:    caller.Did(),
			DIDDoc: caller.DOC(),
		},
		// when out-of-bound and did-exchange protocols are supported we
		// should start to save connection_id to Thread.PID
		Thread: &decorator.Thread{ID: deTask.InvitationID},
	})

	addToSovCacheIf(ssiWA, caller)

	// Save needed data to PSM related Pairwise Representative
	pwr := &pairwiseRep{
		StateKey:   psm.StateKey{DID: me, Nonce: deTask.ID()},
		Name:       deTask.ID(),
		TheirLabel: deTask.Invitation.Label,
		Caller:     didRep{DID: caller.Did(), VerKey: caller.VerKey(), My: true},
		Callee:     didRep{},
	}
	try.To(psm.AddRep(pwr))

	// Create payload to send
	opl := aries.PayloadCreator.NewMsg(task.ID(), pltype.AriesConnectionRequest, msg)

	// Create secure pipe to send payload to other end of the new PW
	receiverKey := task.ReceiverEndp().Key
	receiverKeys := buildRouting(task.ReceiverEndp().Endp, receiverKey,
		deTask.Invitation.RoutingKeys, didMethod)
	callee := try.To1(wa.NewOutDID(receiverKeys...))
	secPipe := sec.Pipe{In: caller, Out: callee}
	wa.AddPipeToPWMap(secPipe, pwr.Name)

	// Update PSM state, and send the payload to other end
	try.To(prot.UpdatePSM(me, connectionID, task, opl, psm.Sending))
	try.To(comm.SendPL(secPipe, task, opl))

	// Sending went OK, update PSM once again
	wpl := aries.PayloadCreator.New(
		didcomm.PayloadInit{
			ID:   task.ID(),
			Type: pltype.AriesConnectionResponse,
		})
	try.To(prot.UpdatePSM(me, connectionID, task, wpl, psm.Waiting))
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
	defer err2.Annotate("connection req", &err)

	// The agent DID, the PW DID is msgMeDID below
	meDID := packet.Receiver.MyDID().Did()

	ipl := packet.Payload
	cnxAddr := packet.Address
	receiver := packet.Receiver

	safeThreadID := ipl.ThreadID()
	connectionID := cnxAddr.ConnID

	req := ipl.MsgHdr().FieldObj().(*didexchange.Request)
	senderEP := service.Addr{
		Endp: common.Service(req.Connection.DIDDoc, 0).ServiceEndpoint,
		Key:  common.RecipientKeys(req.Connection.DIDDoc, 0)[0],
	}
	receiverEP := cnxAddr.AE()
	task := &comm.TaskBase{
		TaskHeader: comm.TaskHeader{
			TaskID:   ipl.ThreadID(),
			TypeID:   ipl.Type(),
			Receiver: receiverEP,
			Sender:   senderEP,
		},
	}

	try.To(prot.UpdatePSM(meDID, connectionID, task, ipl, psm.Received))

	task.SwitchDirection()

	// MARK: we must switch the Nonce for pairwise construction. We will return
	//  it back after we are done. This is because AcaPy compatibility
	ipl.MsgHdr().Thread().ID = connectionID

	wca := receiver.(ssi.Agent)
	calleePw := pairwise.NewCalleePairwise(
		didexchange.ResponseCreator, wca, ipl.MsgHdr().(didcomm.PwMsg))

	calleePw.CheckPreallocation(cnxAddr)

	msg := try.To1(calleePw.ConnReqToRespWithSet(func(m didcomm.PwMsg) {
		// Legacy: calculate our endpoint for the pairwise
		// when DIDDoc is used for all the DIDs extra info.

		pubEndp := *cnxAddr           // set our agent's URL as a base addr
		pubEndp.ConnID = connectionID // set our pw DID to actual agent DID in addr
		pubEndp.VerKey = m.VerKey()   // set our pw VerKey as well

		m.SetEndpoint(service.Addr{
			Endp: pubEndp.Address(),
			Key:  pubEndp.VerKey,
		})
	}))

	// MARK: we must switch the Nonce for pairwise construction back. NOW we
	//  return it back after we are done. This is because AcaPy compatibility
	ipl.MsgHdr().Thread().ID = safeThreadID
	// MARK: very very important to rollback this as well
	glog.V(1).Infoln("=== msg.Thread.ID", msg.Thread().ID, safeThreadID)
	msg.Thread().ID = safeThreadID

	IncomingPWMsg := ipl.MsgHdr().(didcomm.PwMsg) // incoming pairwise message
	caller := calleePw.Caller                     // the other end, we're here the callee
	callerEndp := endp.NewAddrFromPublic(IncomingPWMsg.Endpoint())
	callerAddress := callerEndp.Address()
	pwr := &pairwiseRep{
		StateKey:   psm.StateKey{DID: meDID, Nonce: safeThreadID}, // check if this really must be connection id
		Name:       connectionID,
		TheirLabel: req.Label,
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

	res := msg.FieldObj().(*didexchange.Response)
	try.To(signature.Sign(res, pipe)) // we must sign the Response before send it

	caller.SetAEndp(IncomingPWMsg.Endpoint())
	receiver.AddToPWMap(calleePw.Callee, caller, connectionID) // to access PW later, map it

	// build the response payload, update PSM, and send the PL with sec.Pipe
	opl := aries.PayloadCreator.NewMsg(utils.UUID(), pltype.AriesConnectionResponse, msg)
	try.To(prot.UpdatePSM(meDID, connectionID, task, opl, psm.Sending))
	try.To(comm.SendPL(pipe, task, opl))

	// update the PSM, we are ready at this end for this protocol
	emptyMsg := aries.MsgCreator.Create(didcomm.MsgInit{})
	wpl := aries.PayloadCreator.NewMsg(task.ID(), pltype.AriesConnectionResponse, emptyMsg)
	try.To(prot.UpdatePSM(meDID, connectionID, task, wpl, psm.ReadyACK))

	return nil
}

func handleConnectionResponse(packet comm.Packet) (err error) {
	defer err2.Annotate("connection response", &err)

	connectionID := packet.Address.ConnID
	meDID := packet.Receiver.MyDID().Did()
	ipl := packet.Payload
	receiver := packet.Receiver

	nonce := ipl.ThreadID()
	response := ipl.MsgHdr().FieldObj().(*didexchange.Response)
	if !try.To1(signature.Verify(response)) {
		glog.Error("cannot verify Connection Response signature --> send NACK")
		return errors.New("cannot verify connection response signature")
		// todo: send NACK here
	}

	respEndp := response.Endpoint()
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

	im := ipl.MsgHdr().(didcomm.PwMsg)

	// Set pairwise info about other end to wallet
	var callee core.DID
	if method.TypePeer == utils.Settings.DIDMethod() {
		callee = receiver.LoadDID(im.Did())
	} else { // default method is did:sov:
		callee = ssi.NewDid(im.Did(), im.VerKey())
	}

	callee.Store(receiver.ManagedWallet())

	pwName := pwr.Name
	route := didexchange.RouteForConnection(response.Connection)
	caller.SavePairwiseForDID(managedStorage(receiver), callee, core.PairwiseMeta{
		Name:  pwName,
		Route: route,
	})

	// SAVE ENDPOINT to wallet
	calleeEndp := endp.NewAddrFromPublic(im.Endpoint())
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

	callee.SetAEndp(im.Endpoint())
	receiver.AddToPWMap(caller, callee, pwName) // to access PW later, map it

	// Update that PSM is successfully Ready
	emptyMsg := aries.MsgCreator.Create(didcomm.MsgInit{})
	opl := aries.PayloadCreator.NewMsg(utils.UUID(), pltype.AriesConnectionResponse, emptyMsg)
	try.To(prot.UpdatePSM(meDID, connectionID, task, opl, psm.ReadyACK))

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

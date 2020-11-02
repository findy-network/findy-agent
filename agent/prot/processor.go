package prot

import (
	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/sec"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

// Transition is a Protocol State Machine transition definition. It combines
// rules to execute state transition i.e. move to next state.
type Transition struct {
	comm.Packet
	SendNext    string // the type of the PL we will send next if any
	WaitingNext string // the type of the PL we will wait if any
	SendOnNACK  string // the type to send when we NACK
	InOut              // the handler func, NOTE! return false in all NACK cases
}

// TransitionHandler is a type for Transition to process PSM state transition.
// It receivers input msg and produce output msg. Implementor should return
// false if it wants to NACK otherwise true.
type InOut func(im, om didcomm.MessageHdr) (ack bool, err error)

// Initial is a PSM starting config. It will init PSM accordingly. What msg send
// next and what message wait for. It has Save handler where PSM persistence can
// be handled
type Initial struct {
	SendNext    string        // the type of the PL we will send next if any
	WaitingNext string        // the type of the PL we will wait if any
	Ca          comm.Receiver // the start CA
	T           *comm.Task    // the start TAsk
	Setup                     // setup & save the msg data at the PSM start
}
type Setup func(key psm.StateKey, msg didcomm.MessageHdr) (err error)

type Again struct {
	CA          comm.Receiver // the start CA
	InMsg       didcomm.Msg   // Client data
	SendNext    string        // the type of the PL we will send next if any
	WaitingNext string        // the type of the PL we will wait if any
	SendOnNACK  string        // the type to send when we NACK
	Transfer                  // input/output protocol msg
}

type Transfer func(wa comm.Receiver, im, om didcomm.MessageHdr) (ack bool, err error)

// StartPSM starts the protocol state machine according to Initial struct by
// finally sending the protocol message. During the processing the Initial data,
// the function calls the Save callback where the caller can perform needed
// processing.
func StartPSM(ts Initial) (err error) {
	defer err2.Annotate("aries start PSM", &err)

	wDID := ts.Ca.WDID()
	wa := ts.Ca.WorkerEA()

	defer err2.Handle(&err, func() {
		opl := newPayload(ts)
		_ = UpdatePSM(wDID, "", ts.T, opl, psm.Failure)
	})

	pipe := e2.Pipe.Try(wa.PwPipe(ts.T.Message))
	msgMeDID := pipe.In.Did()
	agentEndp := e2.Public.Try(pipe.EA())
	ts.T.ReceiverEndp = agentEndp

	msg := aries.MsgCreator.Create(didcomm.MsgInit{
		Type:   ts.SendNext,
		Thread: decorator.NewThread(ts.T.Nonce, ""),
		Info:   ts.T.Info,
		ID:     ts.T.ID,
	})

	// Let caller of StartPSM() to update T data so that it can set what we'll
	// send to receiver inside the PL.Message. So this must be here before we
	// Encrypt and seal the output message (om) into PL
	err2.Check(ts.Setup(psm.StateKey{DID: wDID, Nonce: ts.T.Nonce}, msg))

	opl := aries.PayloadCreator.NewMsg(ts.T.Nonce, ts.SendNext, msg)

	err2.Check(UpdatePSM(wDID, msgMeDID, ts.T, opl, psm.Sending))
	err2.Check(comm.SendPL(pipe, ts.T, opl))

	// sending went OK, update PSM for what we are doing next: waiting a
	// message from other side or we are ready.
	nextState := psm.Waiting
	if ts.WaitingNext == pltype.Terminate {
		nextState = psm.ReadyACK
	}
	wpl := aries.PayloadCreator.New(
		didcomm.PayloadInit{ID: ts.T.Nonce, Type: ts.WaitingNext})
	err2.Check(UpdatePSM(wDID, msgMeDID, ts.T, wpl, nextState))

	return err
}

func newPayload(ts Initial) didcomm.Payload {
	return aries.PayloadCreator.New(didcomm.PayloadInit{ID: ts.T.Nonce, Type: ts.SendNext})
}

// ContinuePSM continues PSM when, usually user, has answered what do with the
// protocol. According to the Aries protocol spec there are messages that need
// to be verified by user if they can be continued. With this function user's
// decision is given to the PSM. The PSM can continue or it can send NACK to
// other end and terminate. All that's defined in Again struct.
func ContinuePSM(ts Again) (err error) {
	defer err2.Annotate("continue PSM", &err)

	wDID := ts.CA.WDID()
	wa := ts.CA.WorkerEA()

	PSM, err := psm.GetPSM(psm.StateKey{
		DID:   wDID,
		Nonce: ts.InMsg.SubLevelID(),
	})
	err2.Check(err)

	t := PSM.TaskFor(ts.InMsg.Info())

	msgMeDID := PSM.InDID
	meDID := PSM.Key.DID

	inDID := wa.LoadDID(msgMeDID)
	outStr, _ := err2.StrStr.Try(wa.FindPW(inDID.Did()))
	outDID := wa.LoadDID(outStr)
	outDID.StartEndp(wa.Wallet())
	pipe := sec.Pipe{In: inDID, Out: outDID}

	sendBack := ts.SendNext != pltype.Terminate
	plType := ts.SendNext
	isLast := ts.WaitingNext == pltype.Terminate
	ackFlag := psm.ACK

	im := aries.MsgCreator.Create(didcomm.MsgInit{
		Nonce: ts.InMsg.SubLevelID(), // Continue Task ID comes in as Msg.ID
		Ready: ts.InMsg.Ready()},     // How we continue comes in Ready field
	)
	om := aries.MsgCreator.Create(didcomm.MsgInit{
		Type:   plType,
		Thread: decorator.NewThread(ts.InMsg.SubLevelID(), ""),
	})

	if !err2.Bool.Try(ts.Transfer(wa, im, om)) { // if handler says NACK
		if ts.SendOnNACK != "" {
			sendBack = true        // set if we'll send NACK
			plType = ts.SendOnNACK // NACK type to send
		}
		isLast = true      // our current system NACK ends the PSM
		ackFlag = psm.NACK // we are terminating PSM with NACK
	}

	if sendBack {
		opl := aries.PayloadCreator.NewMsg(utils.UUID(), plType, om)
		agentEndp := e2.Public.Try(pipe.EA())
		t.ReceiverEndp = agentEndp

		err2.Check(UpdatePSM(meDID, msgMeDID, t, opl, psm.Sending))
		err2.Check(comm.SendPL(pipe, t, opl))
	}
	if isLast {
		wpl := aries.PayloadCreator.New(didcomm.PayloadInit{ID: t.Nonce, Type: plType})
		err2.Check(UpdatePSM(meDID, msgMeDID, t, wpl, psm.Ready|ackFlag))
	} else {
		wpl := aries.PayloadCreator.New(didcomm.PayloadInit{ID: t.Nonce, Type: ts.WaitingNext})
		err2.Check(UpdatePSM(meDID, msgMeDID, t, wpl, psm.Waiting))
	}

	return err
}

// ExecPSM is a generic protocol handler function for PSM transitions. ts
// will guide the the execution. Note! that MHandler should return false in all
// of the NACK cases: when receiving NACK even not responding, and when
// terminating current PSM with NACK.
func ExecPSM(ts Transition) (err error) {
	defer err2.Annotate("PSM transition", &err)

	ackFlag := psm.ACK
	meDID := ts.Receiver.Trans().MessagePipe().In.Did()
	sendBack := ts.SendNext != pltype.Terminate && ts.InOut != nil
	plType := ts.SendNext
	isLast := ts.WaitingNext == pltype.Terminate
	if !sendBack {
		plType = ts.Payload.Type()
	}

	// Task is a helper struct here by gathering all needed data for one unit
	task := comm.NewTaskRawPayload(ts.Payload)

	msgMeDID := ts.Address.RcvrDID

	defer err2.Handle(&err, func() {
		_ = UpdatePSM(meDID, msgMeDID, task, ts.Payload, psm.Failure)
	})

	err2.Check(UpdatePSM(meDID, msgMeDID, task, ts.Payload, psm.Received))

	var om didcomm.MessageHdr
	var ep sec.Pipe
	if ts.InOut != nil {
		inDID := ts.Receiver.LoadDID(ts.Address.RcvrDID)
		outStr, _ := err2.StrStr.Try(ts.Receiver.FindPW(inDID.Did()))
		outDID := ts.Receiver.LoadDID(outStr)
		outDID.StartEndp(ts.Receiver.Wallet())
		ep = sec.Pipe{In: inDID, Out: outDID}
		im := ts.Payload.MsgHdr()

		opl := aries.PayloadCreator.NewMsg(task.Nonce, ts.Payload.Type(), im)
		err2.Check(UpdatePSM(meDID, msgMeDID, task, opl, psm.Decrypted))

		om = aries.MsgCreator.Create(
			didcomm.MsgInit{
				Type:   ts.SendNext,         // if we don't reply, generic Msg is used
				Thread: ts.Payload.Thread(), // very important!
			}).(didcomm.MessageHdr)

		if !err2.Bool.Try(ts.InOut(im, om)) { // if handler says NACK
			if ts.SendOnNACK != pltype.Nothing {
				sendBack = true        // set if we'll send NACK
				plType = ts.SendOnNACK // NACK type to send
			}
			isLast = true      // our current system NACK ends the PSM
			ackFlag = psm.NACK // we are terminating PSM with NACK
		}
	}

	if sendBack && om != nil { // playing safe with nil check
		opl := aries.PayloadCreator.NewMsg(utils.UUID(), plType, om)

		// Get endpoint from secure pipe to save it in case for resending.
		agentEndp := e2.Public.Try(ep.EA())
		task.ReceiverEndp = agentEndp

		err2.Check(UpdatePSM(meDID, msgMeDID, task, opl, psm.Sending))
		err2.Check(comm.SendPL(ep, task, opl))
	}

	if isLast {
		wpl := aries.PayloadCreator.New(didcomm.PayloadInit{ID: task.Nonce, Type: plType})
		err2.Check(UpdatePSM(meDID, msgMeDID, task, wpl, psm.Ready|ackFlag))
	} else {
		wpl := aries.PayloadCreator.New(didcomm.PayloadInit{ID: task.Nonce, Type: ts.WaitingNext})
		err2.Check(UpdatePSM(meDID, msgMeDID, task, wpl, psm.Waiting))
	}
	return nil
}

// starters is a map to start protocols. The key is CA API constant. Note! We
// understand than this is not an ideal solution but at least there are no
// dependency from agent package to individual protocol types.
var starters = map[string]comm.ProtProc{}
var continuators = map[string]comm.ProtProc{}
var statusProviders = map[string]comm.ProtProc{}

// AddStarter adds association between CA API message type and protocol. The
// association is used to start protocol with FindAndStart function.
func AddStarter(t string, proc comm.ProtProc) {
	starters[t] = proc
}

func AddContinuator(t string, proc comm.ProtProc) {
	continuators[t] = proc
}

func AddStatusProvider(t string, proc comm.ProtProc) {
	statusProviders[t] = proc
}

func createTaskForRequest(packet comm.Packet, im, om didcomm.Msg, taskID string, state psm.SubState) *comm.Task {
	om.SetNonce(im.Nonce()) // reply same nonce for the API caller
	// use given task id, or create a new one
	if taskID == "" {
		taskID = utils.UUID()
	}
	om.SetSubLevelID(taskID) // return it to client to monitor

	t := &comm.Task{
		TypeID:       packet.Payload.Type(), // same PL type for new task
		Message:      im.Name(),             // transfer Name to generic message string
		Nonce:        taskID,                // new task ID as nonce
		ID:           im.SubLevelID(),       // additional ..
		Info:         im.Info(),             // .. message data ..
		ReceiverEndp: im.ReceiverEP(),

		ConnectionInvitation: im.ConnectionInvitation(),
		CredDefID:            im.CredDefID(),
		CredentialAttrs:      im.CredentialAttributes(),
		ProofAttrs:           im.ProofAttributes(),
	}
	updatePSM(packet.Receiver, t, state)
	return t
}

func updatePSM(receiver comm.Receiver, t *comm.Task, state psm.SubState) {
	defer err2.Catch(func(err error) {
		glog.Errorf("error in psm update: %s", err)
	})
	msg := aries.MsgCreator.Create(didcomm.MsgInit{
		Type:   t.TypeID,
		Thread: decorator.NewThread(t.Nonce, ""),
	})
	wDID := receiver.WDID()
	opl := aries.PayloadCreator.NewMsg(t.Nonce, t.TypeID, msg)
	err2.Check(UpdatePSM(wDID, "", t, opl, state))
}

// InitTask initialises the task to waiting state
func InitTask(packet comm.Packet, im, om didcomm.Msg) *comm.Task {
	defer err2.CatchTrace(func(err error) {
		glog.Error("Cannot init task")
	})

	t := createTaskForRequest(packet, im, om, "", psm.Waiting)

	return t
}

// FindAndStart start the protocol by using CA API Type in the packet.PL.
func FindAndStart(packet comm.Packet, im, om didcomm.Msg, taskID string) (tID string) {
	defer err2.CatchTrace(func(err error) {
		glog.Error("Cannot start protocol")
	})

	proc, ok := starters[packet.Payload.Type()]
	if !ok {
		s := "!!!! No protocol starter !!!"
		glog.Error(s, packet.Payload.Type())
		panic(s)
	}

	t := createTaskForRequest(packet, im, om, taskID, psm.Sending)

	go proc.Starter(packet.Receiver, t)

	return taskID
}

// FindAndStartTask start the protocol by using CA API Type in the packet.PL.
func FindAndStartTask(receiver comm.Receiver, task *comm.Task) {
	defer err2.CatchTrace(func(err error) {
		glog.Errorf("Cannot start protocol: %s", err)
	})

	proc, ok := starters[task.TypeID]
	if !ok {
		s := "!!!! No protocol starter !!!"
		glog.Error(s, task.TypeID)
		panic(s)
	}
	updatePSM(receiver, task, psm.Sending)
	go proc.Starter(receiver, task)
}

func Continue(packet comm.Packet, im didcomm.Msg) {
	proc, ok := continuators[packet.Payload.Type()]
	if !ok {
		glog.Error("!!No prot continuator for:", packet.Payload.Type())
		panic("no protocol continuator")
	}

	go proc.Continuator(packet.Receiver, im)
}

func Unpause(rcvr comm.Receiver, typeID string, im didcomm.Msg) {
	proc, ok := continuators[typeID]
	if !ok {
		glog.Error("!!No prot continuator for:", typeID)
		panic("no protocol continuator")
	}

	go proc.Continuator(rcvr, im)
}

func GetStatus(protocol string, key *psm.StateKey) interface{} {
	proc, ok := statusProviders[protocol]
	if !ok {
		glog.Error("!!!! No protocol status getter for " + protocol + " !!!")
		panic("no protocol status getter")
	}

	return proc.Status(key.DID, key.Nonce)
}

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
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
)

// Transition is a Protocol State Machine transition definition. It combines
// rules to execute state transition i.e. move to next state.
type Transition struct {
	comm.Packet
	SendNext    string           // the type of the PL we will send next if any
	WaitingNext string           // the type of the PL we will wait if any
	SendOnNACK  string           // the type to send when we NACK
	InOut                        // the handler func, NOTE! return false in all NACK cases
	TaskHeader  *comm.TaskHeader // updated task data
}

// TransitionHandler is a type for Transition to process PSM state transition.
// It receivers input msg and produce output msg. Implementor should return
// false if it wants to NACK otherwise true.
type InOut func(connID string, im, om didcomm.MessageHdr) (ack bool, err error)

// Initial is a PSM starting config. It will init PSM accordingly. What msg send
// next and what message wait for. It has Save handler where PSM persistence can
// be handled
type Initial struct {
	SendNext    string        // the type of the PL we will send next if any
	WaitingNext string        // the type of the PL we will wait if any
	Ca          comm.Receiver // the start CA
	T           comm.Task     // the start TAsk
	Setup                     // setup & save the msg data at the PSM start
}
type Setup func(key psm.StateKey, msg didcomm.MessageHdr) (err error)

type Again struct {
	CA    comm.Receiver // the start CA
	InMsg didcomm.Msg

	SendNext    string // the type of the PL we will send next if any
	WaitingNext string // the type of the PL we will wait if any
	SendOnNACK  string // the type to send when we NACK
	Transfer           // input/output protocol msg
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

	pipe := e2.Pipe.Try(wa.PwPipe(ts.T.ConnectionID()))
	msgMeDID := pipe.In.Did()
	agentEndp := e2.Public.Try(pipe.EA())
	ts.T.SetReceiverEndp(agentEndp)

	msg := aries.MsgCreator.Create(didcomm.MsgInit{
		Type:   ts.SendNext,
		Thread: decorator.NewThread(ts.T.ID(), ""),
	})

	// Let caller of StartPSM() to update T data so that it can set what we'll
	// send to receiver inside the PL.Message. So this must be here before we
	// Encrypt and seal the output message (om) into PL
	err2.Check(ts.Setup(psm.StateKey{DID: wDID, Nonce: ts.T.ID()}, msg))

	opl := aries.PayloadCreator.NewMsg(ts.T.ID(), ts.SendNext, msg)

	err2.Check(UpdatePSM(wDID, msgMeDID, ts.T, opl, psm.Sending))
	err2.Check(comm.SendPL(pipe, ts.T, opl))

	// sending went OK, update PSM for what we are doing next: waiting a
	// message from other side or we are ready.
	nextState := psm.Waiting
	if ts.WaitingNext == pltype.Terminate {
		nextState = psm.ReadyACK
	}
	wpl := aries.PayloadCreator.New(
		didcomm.PayloadInit{ID: ts.T.ID(), Type: ts.WaitingNext})
	err2.Check(UpdatePSM(wDID, msgMeDID, ts.T, wpl, nextState))

	return err
}

func newPayload(ts Initial) didcomm.Payload {
	return aries.PayloadCreator.New(didcomm.PayloadInit{ID: ts.T.ID(), Type: ts.SendNext})
}

// ContinuePSM continues PSM when, usually user, has answered what do with the
// protocol. According to the Aries protocol spec there are messages that need
// to be verified by user if they can be continued. With this function user's
// decision is given to the PSM. The PSM can continue or it can send NACK to
// other end and terminate. All that's defined in Again struct.
func ContinuePSM(shift Again) (err error) {
	defer err2.Annotate("continue PSM", &err)

	wDID := shift.CA.WDID()
	wa := shift.CA.WorkerEA()

	PSM := e2.PSM.Try(psm.GetPSM(psm.StateKey{
		DID:   wDID,
		Nonce: shift.InMsg.SubLevelID(),
	}))

	presentTask := PSM.PresentTask()

	msgMeDID := PSM.ConnDID
	meDID := PSM.Key.DID

	inDID := wa.LoadDID(msgMeDID)

	pairwise, err := wa.FindPWByDID(inDID.Did())
	err2.Check(err)
	assert.D.True(pairwise != nil, "pairwise should not be nil")

	outDID := wa.LoadTheirDID(*pairwise)
	outDID.StartEndp(wa.Wallet())
	pipe := sec.Pipe{In: inDID, Out: outDID}

	sendBack := shift.SendNext != pltype.Terminate
	plType := shift.SendNext
	isLast := shift.WaitingNext == pltype.Terminate
	ackFlag := psm.ACK

	im := aries.MsgCreator.Create(didcomm.MsgInit{
		Nonce: shift.InMsg.SubLevelID(), // Continue Task ID comes in as Msg.ID
		Ready: shift.InMsg.Ready()},     // How we continue comes in Ready field
	)
	om := aries.MsgCreator.Create(didcomm.MsgInit{
		Type:   plType,
		Thread: decorator.NewThread(shift.InMsg.SubLevelID(), ""),
	})

	if !err2.Bool.Try(shift.Transfer(wa, im, om)) { // if handler says NACK
		if shift.SendOnNACK != "" {
			sendBack = true           // set if we'll send NACK
			plType = shift.SendOnNACK // NACK type to send
		}
		isLast = true      // our current system NACK ends the PSM
		ackFlag = psm.NACK // we are terminating PSM with NACK
	}

	if sendBack {
		opl := aries.PayloadCreator.NewMsg(utils.UUID(), plType, om)
		agentEndp := e2.Public.Try(pipe.EA())
		presentTask.SetReceiverEndp(agentEndp)

		err2.Check(UpdatePSM(meDID, msgMeDID, presentTask, opl, psm.Sending))
		err2.Check(comm.SendPL(pipe, presentTask, opl))
	}
	if isLast {
		wpl := aries.PayloadCreator.New(didcomm.PayloadInit{ID: presentTask.ID(), Type: plType})
		err2.Check(UpdatePSM(meDID, msgMeDID, presentTask, wpl, psm.Ready|ackFlag))
	} else {
		wpl := aries.PayloadCreator.New(didcomm.PayloadInit{ID: presentTask.ID(), Type: shift.WaitingNext})
		err2.Check(UpdatePSM(meDID, msgMeDID, presentTask, wpl, psm.Waiting))
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
	meDID := ts.Receiver.MyDID().Did()
	sendBack := ts.SendNext != pltype.Terminate && ts.InOut != nil
	plType := ts.SendNext
	isLast := ts.WaitingNext == pltype.Terminate
	if !sendBack {
		plType = ts.Payload.Type()
	}

	// Task is a helper struct here by gathering all needed data for one unit
	if ts.TaskHeader == nil {
		ts.TaskHeader = &comm.TaskHeader{}
	}
	ts.TaskHeader.TaskID = ts.Payload.ThreadID()
	ts.TaskHeader.TypeID = ts.Payload.Type()

	// Create protocol task in protocol implementation
	task, err := CreateTask(ts.TaskHeader, nil)
	err2.Check(err)

	msgMeDID := ts.Address.RcvrDID

	defer err2.Handle(&err, func() {
		_ = UpdatePSM(meDID, msgMeDID, task, ts.Payload, psm.Failure)
	})

	err2.Check(UpdatePSM(meDID, msgMeDID, task, ts.Payload, psm.Received))

	var om didcomm.MessageHdr
	var ep sec.Pipe
	if ts.InOut != nil {
		inDID := ts.Receiver.LoadDID(ts.Address.RcvrDID)

		pairwise, err := ts.Receiver.FindPWByDID(inDID.Did())
		err2.Check(err)
		assert.D.True(pairwise != nil, "pairwise should not be nil")

		connID := pairwise.Meta.Name
		outDID := ts.Receiver.LoadTheirDID(*pairwise)
		outDID.StartEndp(ts.Receiver.Wallet())

		ep = sec.Pipe{In: inDID, Out: outDID}
		im := ts.Payload.MsgHdr()

		opl := aries.PayloadCreator.NewMsg(task.ID(), ts.Payload.Type(), im)
		err2.Check(UpdatePSM(meDID, msgMeDID, task, opl, psm.Decrypted))

		om = aries.MsgCreator.Create(
			didcomm.MsgInit{
				Type:   ts.SendNext,         // if we don't reply, generic Msg is used
				Thread: ts.Payload.Thread(), // very important!
			})

		if !err2.Bool.Try(ts.InOut(connID, im, om)) { // if handler says NACK
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
		task.SetReceiverEndp(agentEndp)

		err2.Check(UpdatePSM(meDID, msgMeDID, task, opl, psm.Sending))
		err2.Check(comm.SendPL(ep, task, opl))
	}

	if isLast {
		wpl := aries.PayloadCreator.New(didcomm.PayloadInit{ID: task.ID(), Type: plType})
		err2.Check(UpdatePSM(meDID, msgMeDID, task, wpl, psm.Ready|ackFlag))
	} else {
		wpl := aries.PayloadCreator.New(didcomm.PayloadInit{ID: task.ID(), Type: ts.WaitingNext})
		err2.Check(UpdatePSM(meDID, msgMeDID, task, wpl, psm.Waiting))
	}
	return nil
}

// starters is a map to start protocols. The key is CA API constant. Note! We
// understand than this is not an ideal solution but at least there are no
// dependency from agent package to individual protocol types.
var creators = map[string]comm.ProtProc{}
var starters = map[string]comm.ProtProc{}
var continuators = map[string]comm.ProtProc{}
var statusProviders = map[string]comm.ProtProc{}

// AddStarter adds association between CA API message type and protocol. The
// association is used to start protocol with FindAndStart function.
func AddCreator(t string, proc comm.ProtProc) {
	creators[t] = proc
}

func AddStarter(t string, proc comm.ProtProc) {
	starters[t] = proc
}

func AddContinuator(t string, proc comm.ProtProc) {
	continuators[t] = proc
}

func AddStatusProvider(t string, proc comm.ProtProc) {
	statusProviders[t] = proc
}

func updatePSM(receiver comm.Receiver, t comm.Task, state psm.SubState) {
	defer err2.Catch(func(err error) {
		glog.Errorf("error in psm update: %s", err)
	})
	msg := aries.MsgCreator.Create(didcomm.MsgInit{
		Type:   t.Type(),
		Thread: decorator.NewThread(t.ID(), ""),
	})
	wDID := receiver.WDID()
	opl := aries.PayloadCreator.NewMsg(t.ID(), t.Type(), msg)
	err2.Check(UpdatePSM(wDID, "", t, opl, state))
}

func CreateTask(header *comm.TaskHeader, protocol *pb.Protocol) (t comm.Task, err error) {
	defer err2.Return(&err)

	protocolType := aries.ProtocolForType(header.TypeID)
	taskCreator, ok := creators[protocolType]
	if !ok {
		s := "!!!! No task creator !!! %s, %s"
		glog.Errorf(s, protocolType, header.TypeID)
		panic(s)
	}

	return taskCreator.Creator(header, protocol)
}

// FindAndStartTask start the protocol by using CA API Type in the packet.PL.
func FindAndStartTask(receiver comm.Receiver, task comm.Task) {
	defer err2.CatchTrace(func(err error) {
		glog.Errorf("Cannot start protocol: %s", err)
	})

	proc, ok := starters[task.Type()]
	if !ok {
		s := "!!!! No protocol starter !!!"
		glog.Error(s, task.Type())
		panic(s)
	}
	updatePSM(receiver, task, psm.Sending)
	go proc.Starter(receiver, task)
}

func Resume(rcvr comm.Receiver, typeID, protocolID string, ack bool) {
	proc, ok := continuators[typeID]
	if !ok {
		glog.Error("!!No prot continuator for:", typeID)
		panic("no protocol continuator")
	}

	om := aries.MsgCreator.Create(didcomm.MsgInit{
		Ready: ack,
		ID:    protocolID, // This Has the SubLevelID() Getter
		Nonce: protocolID, // This makes the Thread decorator
	}).(didcomm.Msg)

	go proc.Continuator(rcvr, om)
}

func FillStatus(protocol string, key psm.StateKey, ps *pb.ProtocolStatus) *pb.ProtocolStatus {
	proc, ok := statusProviders[protocol]
	if !ok {
		glog.Error("!!!! No protocol status getter for " + protocol + " !!!")
		panic("no protocol status getter")
	}

	return proc.FillStatus(key.DID, key.Nonce, ps)
}

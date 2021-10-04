package prot

import (
	"fmt"
	"strings"
	"time"

	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/utils"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

type notifyEdge struct {
	did       string // worker agent DID
	plType    string // notification message type id
	nonce     string // protocol ID (not a Aries message ID but thead ID)
	timestamp int64  // the timestamp of the PSM
	pwName    string // connection ID (note!! not a pairwise Label)

	// TODO: change the name protocol starter and moved initiator and addressee
	// other ways
	initiator bool // true if we are to one who started the protocol
}

// NotifyEdge sends notification to CA's controllores.
func NotifyEdge(ne notifyEdge) {
	r := comm.ActiveRcvrs.Get(ne.did)
	if r != nil {
		myCA := r.MyCA()

		go func() {
			defer err2.CatchTrace(func(err error) {
				glog.Warningf("=======\n%s\n=======", err)
			})

			// todo: when CallEA() changes or removes we should get this
			// information from some where performant place
			taskStatus := StatusForTask(ne.did, ne.nonce)

			bus.WantAllAgentActions.AgentBroadcast(bus.AgentNotify{
				AgentKeyType:     bus.AgentKeyType{AgentDID: ne.did},
				ID:               utils.UUID(),
				NotificationType: ne.plType,
				ProtocolID:       ne.nonce,
				ProtocolFamily:   taskStatus.Type,
				ConnectionID:     ne.pwName,
				Timestamp:        ne.timestamp,
				Initiator:        ne.initiator,
			})

			msg := mesg.MsgCreator.Create(didcomm.MsgInit{
				Nonce: ne.nonce,
				Name:  ne.pwName,
				Body:  taskStatus,
			}).(didcomm.Msg)

			// Webhook - catch and ignore errors in response parsing
			_, _ = myCA.CallEA(ne.plType, msg)
		}()
	} else {
		glog.Warning("unable to notify edge with did", ne.did)
	}
}

// UpdatePSM adds new sub state to PSM with timestamp and all the working data.
// The PSM key is meDID (worker agent) and the task.Nonce. The PSM includes all
// state history.
//  meDID = handling agent DID i.e. worker agent DID
//  msgMe = our end's connection aka pairwise DID (important!)
//  task  = current comm.Task struct for additional protocol information
//  opl   = output payload we are building to send, state by state
//  subs  = current sub state of the protocol state machine (PSM)
func UpdatePSM(meDID, msgMe string, task comm.Task, opl didcomm.Payload, subs psm.SubState) (err error) {
	defer err2.Annotate("create psm", &err)

	if glog.V(5) {
		glog.Infof("-- %s->%s[%s:%s]",
			strings.ToUpper(opl.ProtocolMsg()), subs, meDID, task.ID())
	}

	machineKey := psm.StateKey{DID: meDID, Nonce: task.ID()}

	// NOTE!!! We cannot use error handling with the GetPSM because it reports
	// not founding as an error. TODO: It must be fixed. Filtering errors by
	// their values is a mistake, it brings more dependencies.
	m, _ := psm.GetPSM(machineKey)

	var machine *psm.PSM
	timestamp := time.Now().UnixNano()
	s := psm.State{
		Timestamp: timestamp,
		T:         task,
		PLInfo:    psm.PayloadInfo{Type: opl.Type()},
		Sub:       subs,
	}
	if m != nil { // update existing one
		if msgMe != "" {
			m.InDID = msgMe
		}
		m.States = append(m.States, s)
		machine = m
	} else { // create a new one
		ss := make([]psm.State, 1, 12)
		ss[0] = s

		initiator := false

		if task.Role() != pb.Protocol_UNKNOWN { // let task define role when it is set
			initiator = task.Role() == pb.Protocol_INITIATOR
		} else { // otherwise try to detect role from state
			// TODO: in propose cases the actor starting the flow is currently addressee
			// so the role is reported incorrectly for the "receiving" side
			initiator = subs&(psm.Sending|psm.Failure) != 0
		}

		role := pb.Protocol_INITIATOR.String()
		if initiator {
			role = pb.Protocol_ADDRESSEE.String()
		}
		glog.V(3).Infof("----- We (%s) are %s (%s) ----", meDID, role, task.Role())

		machine = &psm.PSM{Key: machineKey, InDID: msgMe,
			States: ss, Initiator: initiator}
	}
	err2.Check(psm.AddPSM(machine))

	plType := opl.Type()
	if plType == pltype.Nothing {
		plType = machine.FirstState().PLInfo.Type
	}

	// TODO: add machine to endingInfo to allow 'cheap' data access for
	// notifications
	go triggerEnd(endingInfo{
		timestamp:         timestamp,
		subState:          subs,
		nonce:             task.ID(),
		meDID:             meDID,
		pwName:            machine.PairwiseName(),
		plType:            plType,
		pendingUserAction: machine.PendingUserAction(),
		initiator:         machine.Initiator,
	})

	return nil
}

// AddFlagUpdatePSM updates existing PSM by adding a sub-state with state flag:
//  lastSubState | subState  => adding a new sub state flag to last one
// and if needed sub-state can be cleared before adding a new one:
//  lastSubState = lastSubState ^& unsetSubState
func AddAndSetFlagUpdatePSM(
	machineKey psm.StateKey,
	subState psm.SubState,
	unsetSubState psm.SubState) (err error) {

	defer err2.Annotate("mark archive psm", &err)

	// NOTE!!! We cannot use error handling with the GetPSM because it reports
	// not founding as an error. TODO: It must be fixed. Filtering errors by
	// their values is a mistake, it brings more dependencies.
	m, _ := psm.GetPSM(machineKey)

	clearedLastSubState := m.LastState().Sub &^ unsetSubState
	var machine *psm.PSM
	timestamp := time.Now().UnixNano()
	s := psm.State{Timestamp: timestamp, Sub: subState | clearedLastSubState}
	if m != nil { // update existing one
		m.States = append(m.States, s)
		machine = m
	} else {
		return fmt.Errorf("previous PSM (%s) must exist", machineKey)
	}
	err2.Check(psm.AddPSM(machine))

	if subState&(psm.Archiving|psm.Archived) != 0 {
		go notifyArchiving(machine, endingInfo{
			timestamp:         timestamp,
			subState:          subState,
			nonce:             machineKey.Nonce,
			meDID:             machineKey.DID,
			pwName:            machine.PairwiseName(),
			plType:            machine.FirstState().T.Type(),
			pendingUserAction: machine.PendingUserAction(),
			initiator:         machine.Initiator,
		})
	}
	return nil
}

func notifyArchiving(machine *psm.PSM, info endingInfo) {
	key := psm.StateKey{
		DID:   info.meDID,
		Nonce: info.nonce,
	}
	notify := bus.AgentNotify{
		AgentKeyType:     bus.AgentKeyType{AgentDID: key.DID},
		ID:               utils.UUID(),
		NotificationType: info.plType,
		ProtocolID:       info.nonce,
		ProtocolFamily:   machine.Protocol(),
		ConnectionID:     info.pwName,
		Timestamp:        info.timestamp,
		Initiator:        info.initiator,
	}
	if info.subState&psm.Archiving != 0 {
		glog.V(1).Infoln("archiving:", key)
		bus.WantAllAgencyActions.AgentBroadcast(notify)
	} else if info.subState&psm.Archived != 0 {
		glog.V(1).Infoln("**** ARCHIVED:", key)
		bus.WantAllPSMCleanup.AgentBroadcast(notify)
	}

}

type endingInfo struct {
	subState          psm.SubState
	nonce             string
	meDID             string
	pwName            string
	plType            string
	timestamp         int64
	pendingUserAction bool
	initiator         bool
}

func triggerEnd(info endingInfo) {
	defer err2.CatchTrace(func(err error) {
		glog.Error("trigger PSM end notification:", err)
	})

	key := psm.StateKey{
		DID:   info.meDID,
		Nonce: info.nonce,
	}

	switch info.subState.Pure() {
	case psm.Ready:
		// Do broadcasts and cleanup, this machine is ready
		ack := info.subState&psm.ACK != 0
		if ack {
			if info.plType == pltype.Nothing {
				glog.Warning("PL type is empty on Notify")
			}
			NotifyEdge(notifyEdge{
				did:       info.meDID,
				plType:    pltype.CANotifyStatus,
				nonce:     info.nonce,
				timestamp: info.timestamp,
				pwName:    info.pwName,
				initiator: info.initiator,
			})
		}
	case psm.Waiting:
		// Notify also tasks that are waiting for user action
		if info.pendingUserAction {
			bus.WantUserActions.Broadcast(key, info.subState)
			NotifyEdge(notifyEdge{
				did:       info.meDID,
				plType:    pltype.CANotifyUserAction,
				nonce:     info.nonce,
				timestamp: info.timestamp,
				pwName:    info.pwName,
				initiator: info.initiator,
			})
		}
	}
	// To brave one who wants to know all
	bus.WantAll.Broadcast(key, info.subState)
}

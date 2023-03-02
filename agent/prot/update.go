package prot

import (
	"fmt"
	"strings"
	"time"

	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/utils"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

type notifyEdge struct {
	did       string // worker agent DID
	plType    string // notification message type id
	nonce     string // protocol ID (not a Aries message ID but thead ID)
	timestamp int64  // the timestamp of the PSM
	pwName    string // connection ID (note!! not a pairwise Label)
	family    string // protocol family

	// startedByUs if we are the one who sent the first message
	startedByUs bool

	role pb.Protocol_Role
}

// NotifyEdge sends notification to CA's controllores.
func NotifyEdge(ne notifyEdge) {
	r := comm.ActiveRcvrs.Get(ne.did)
	if r != nil {
		go func() {
			defer err2.Catch(func(err error) {
				glog.Warningf("=======\n%s\n=======", err)
			})

			bus.WantAllAgentActions.AgentBroadcast(bus.AgentNotify{
				AgentKeyType:     bus.AgentKeyType{AgentDID: ne.did},
				ID:               utils.UUID(),
				NotificationType: ne.plType,
				ProtocolID:       ne.nonce,
				ProtocolFamily:   ne.family,
				ConnectionID:     ne.pwName,
				Timestamp:        ne.timestamp,
				Role:             ne.role,
			})
		}()
	} else {
		glog.Warning("unable to notify edge with did", ne.did)
	}
}

// UpdatePSM adds new sub state to PSM with timestamp and all the working data.
// The PSM key is meDID (worker agent) and the task.Nonce. The PSM includes all
// state history.
//
//	meDID = handling agent DID i.e. worker agent DID
//	connID = connection ID
//	task  = current comm.Task struct for additional protocol information
//	opl   = output payload we are building to send, state by state
//	subs  = current sub state of the protocol state machine (PSM)
func UpdatePSM(
	agentDID,
	connID string,
	task comm.Task,
	opl didcomm.Payload,
	stateType psm.SubState,
) (
	err error,
) {
	defer err2.Handle(&err, "create psm")

	if glog.V(5) {
		glog.Infof("-- %s->%s[%s:%s]",
			strings.ToUpper(opl.ProtocolMsg()), stateType, agentDID, task.ID())
	}

	PSMKey := psm.StateKey{DID: agentDID, Nonce: task.ID()}
	foundPSM := try.To1(psm.FindPSM(PSMKey))

	var currentPSM *psm.PSM
	timestamp := time.Now().UnixNano()
	currentState := psm.State{
		Timestamp: timestamp,
		T:         task,
		PLInfo:    psm.PayloadInfo{Type: opl.Type()},
		Sub:       stateType,
	}
	if foundPSM != nil { // update existing one
		if !foundPSM.Accept(stateType) {
			glog.Warningf("PSM doesn't acccept %v -> %v. Skipping..",
				foundPSM.LastState().Sub, stateType)
			return nil
		}
		if connID != "" {
			foundPSM.ConnID = connID
		}
		foundPSM.States = append(foundPSM.States, currentState)
		currentPSM = foundPSM
	} else { // create a new one
		states := make([]psm.State, 1, 12)
		states[0] = currentState

		startedByUs := true
		role := task.Role()

		if task.Role() == pb.Protocol_UNKNOWN {
			startedByUs = false
			role = pltype.ProtocolRoleForType(opl.ProtocolMsg())
		}

		glog.V(3).Infof("----- We (send by us: %v) are %s (%s) ----",
			startedByUs,
			agentDID,
			role,
		)

		currentPSM = &psm.PSM{
			Key:         PSMKey,
			ConnID:      connID,
			States:      states,
			StartedByUs: startedByUs,
			Role:        role,
		}
	}
	try.To(psm.AddPSM(currentPSM))

	plType := opl.Type()
	if plType == pltype.Nothing {
		plType = currentPSM.FirstState().PLInfo.Type
	}

	// TODO: add machine to endingInfo to allow 'cheap' data access for
	// notifications, WIP: adding protocolFamily as first step
	go triggerEnd(endingInfo{
		timestamp:         timestamp,
		subState:          stateType,
		nonce:             task.ID(),
		meDID:             agentDID,
		pwName:            connID,
		plType:            plType,
		pendingUserAction: currentPSM.PendingUserAction(),
		startedByUs:       currentPSM.StartedByUs,
		userActionType:    task.UserActionType(),
		protocolFamily:    currentPSM.Protocol(),
		role:              currentPSM.Role,
	})

	return nil
}

// AddFlagUpdatePSM updates existing PSM by adding a sub-state with state flag:
//
//	lastSubState | subState  => adding a new sub state flag to last one
//
// and if needed sub-state can be cleared before adding a new one:
//
//	lastSubState = lastSubState ^& unsetSubState
func AddAndSetFlagUpdatePSM(
	machineKey psm.StateKey,
	subState psm.SubState,
	unsetSubState psm.SubState) (err error) {

	defer err2.Handle(&err, "mark archive psm")

	m := try.To1(psm.GetPSM(machineKey))

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
	try.To(psm.AddPSM(machine))

	if subState&(psm.Archiving|psm.Archived) != 0 {
		go notifyArchiving(machine, endingInfo{
			timestamp:         timestamp,
			subState:          subState,
			nonce:             machineKey.Nonce,
			meDID:             machineKey.DID,
			pwName:            m.ConnID,
			plType:            machine.FirstState().T.Type(),
			pendingUserAction: machine.PendingUserAction(),
			startedByUs:       machine.StartedByUs,
			role:              machine.Role,
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
		Role:             machine.Role,
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
	startedByUs       bool
	userActionType    string
	protocolFamily    string
	role              pb.Protocol_Role
}

func triggerEnd(info endingInfo) {
	defer err2.Catch(func(err error) {
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
				did:         info.meDID,
				plType:      pltype.CANotifyStatus,
				nonce:       info.nonce,
				timestamp:   info.timestamp,
				pwName:      info.pwName,
				family:      info.protocolFamily,
				startedByUs: info.startedByUs,
				role:        info.role,
			})
		}
	case psm.Waiting, psm.Failure:
		plType := pltype.Nothing
		// Notify tasks that are waiting for user action
		if info.pendingUserAction {
			plType = info.userActionType
		}
		// ...or failed
		if info.subState&psm.Failure != 0 {
			plType = pltype.CANotifyStatus
		}
		if plType != pltype.Nothing {
			bus.WantUserActions.Broadcast(key, info.subState)
			NotifyEdge(notifyEdge{
				did:         info.meDID,
				plType:      plType,
				nonce:       info.nonce,
				timestamp:   info.timestamp,
				pwName:      info.pwName,
				family:      info.protocolFamily,
				startedByUs: info.startedByUs,
				role:        info.role,
			})
		}
	}
	// To brave one who wants to know all
	bus.WantAll.Broadcast(key, info.subState)
}

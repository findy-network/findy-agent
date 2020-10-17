package prot

import (
	"strings"
	"time"

	"github.com/findy-network/findy-agent/agent/apns"
	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

// NotifyEdge sends notification to client (previously edge agent). It sends
// notifications via apns, web socket, and web hook if any of these are
// available.
//  did      = worker agent DID
//  plType   = notification message type id
//  nonce    = protocol ID (not a Aries message ID but thead ID)
//  pwName   = connection ID (note!! not a pairwise Label)
func NotifyEdge(did, plType, nonce, pwName string) {
	r := comm.ActiveRcvrs.Get(did)
	if r != nil {
		myCA := r.MyCA()
		go func() {
			defer err2.CatchTrace(func(err error) {
				glog.Warningf("=======\n%s\n=======", err)
			})
			apns.Push(did)

			taskStatus := StatusForTask(did, nonce)

			bus.WantAllAgentActions.AgentBroadcast(bus.AgentNotify{
				AgentKeyType:     bus.AgentKeyType{AgentDID: did},
				NotificationType: plType,
				ProtocolID:       nonce,
				ProtocolFamily:   taskStatus.Type,
				ConnectionID:     pwName,
				TimestampMs:      taskStatus.TimestampMs,
			})

			msg := mesg.MsgCreator.Create(didcomm.MsgInit{
				Nonce: nonce,
				Name:  pwName,
				Body:  taskStatus,
			}).(didcomm.Msg)

			// Websocket
			myCA.NotifyEA(plType, msg)
			// Webhook - catch and ignore errors in response parsing
			_, _ = myCA.CallEA(plType, msg)
		}()
	} else {
		glog.Warning("unable to notify edge with did", did)
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
func UpdatePSM(meDID, msgMe string, task *comm.Task, opl didcomm.Payload, subs psm.SubState) (err error) {
	defer err2.Annotate("create psm", &err)

	if glog.V(5) {
		glog.Infof("-- %s->%s[%s:%s]",
			strings.ToUpper(opl.ProtocolMsg()), subs, meDID, task.Nonce)
	}

	machineKey := psm.StateKey{DID: meDID, Nonce: task.Nonce}
	m, _ := psm.GetPSM(machineKey)

	var machine *psm.PSM
	s := psm.State{Timestamp: time.Now().UnixNano(), T: *task, PLInfo: psm.PayloadInfo{Type: opl.Type()}, Sub: subs}
	if m != nil { // update existing one
		if msgMe != "" {
			m.InDID = msgMe
		}
		m.States = append(m.States, s)
		machine = m
	} else { // create a new one
		ss := make([]psm.State, 1, 12)
		ss[0] = s
		machine = &psm.PSM{Key: machineKey, InDID: msgMe, States: ss}
	}
	err2.Check(psm.AddPSM(machine))

	plType := opl.Type()
	if plType == pltype.Nothing {
		plType = machine.FirstState().PLInfo.Type
	}
	go triggerEnd(endingInfo{
		subState:          subs,
		nonce:             task.Nonce,
		meDID:             meDID,
		pwName:            machine.PairwiseName(),
		plType:            plType,
		pendingUserAction: machine.PendingUserAction(),
	})

	return nil
}

type endingInfo struct {
	subState          psm.SubState
	nonce             string
	meDID             string
	pwName            string
	plType            string
	pendingUserAction bool
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
			NotifyEdge(info.meDID, pltype.CANotifyStatus, info.nonce, info.pwName)
		}
		bus.ReadyStation.BroadcastReady(key, ack)
	case psm.Failure:
		// Do broadcasts for chained protocols to be able to report clients
		bus.ReadyStation.BroadcastReady(key, false)
	case psm.Waiting:
		// Notify also tasks that are waiting for user action
		if info.pendingUserAction {
			bus.WantUserActions.Broadcast(key, info.subState)
			NotifyEdge(info.meDID, pltype.CANotifyUserAction, info.nonce, info.pwName)
		}
	}
	// To brave one who wants to know all
	bus.WantAll.Broadcast(key, info.subState)
}

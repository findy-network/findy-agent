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

func NotifyEdge(did, plType, nonce, pwName string) {
	r := comm.ActiveRcvrs.Get(did)
	if r != nil {
		myCA := r.MyCA()
		go func() {
			defer err2.CatchTrace(func(err error) {
				glog.Warning(err)
			})
			apns.Push(did)

			msg := mesg.MsgCreator.Create(didcomm.MsgInit{
				Nonce: nonce,
				Name:  pwName,
				Body:  StatusForTask(did, nonce),
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
			NotifyEdge(info.meDID, pltype.CANotifyUserAction, info.nonce, info.pwName)
		}
	}
	// To brave one who wants to know all
	bus.Broadcast(key, info.subState)
}

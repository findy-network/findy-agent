package psm

import (
	"log"

	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-wrapper-go/dto"
)

/*
@startuml
title SubState

[*] -> waiting
waiting -> received
received: do/decrypt
received -> decrypted
decrypted -> sending
sending --> failure: IO/handler\nerror
sending: enter/**call handler**
sending: do/process IO
received -> failure: decrypt\nerror
sending -> waiting: send OK
sending -> ready: was last ACK
sending -> ready: __did__ receive ACK/NACK
sending -> ready: **handler** says ACK/NACK
failure -> [*]
ready --> [*]
state ready {
	[*] --> ACK
	[*] --> NACK
	ACK --> [*]
	NACK --> [*]

}
@enduml
*/

// SubState is enumeration for the state transitions PSM will have during its
// execution. The above PUML diagram illustrates what transitions are currently
// recognized. The Ready state should have 2 internal states: ACK/NACK
type SubState uint

// States of the PSM`s individual state can be.
const (
	ACK  SubState = 0x01 << iota // Sub sub state of Ready
	NACK                         // Sub sub state of Ready
	Waiting
	Received
	Decrypted
	Sending
	Ready
	Failure
)

const (
	ReadyACK  SubState = Ready | ACK
	ReadyNACK SubState = Ready | NACK
)

func (ss SubState) String() string {
	switch ss {
	case ReadyACK:
		return "ReadyACK"
	case ACK:
		return "ACK"
	case ReadyNACK:
		return "ReadyNACK"
	case NACK:
		return "NACK"
	case Ready:
		return "Ready"
	case Waiting:
		return "Waiting"
	case Received:
		return "Received"
	case Decrypted:
		return "Decrypted"
	case Sending:
		return "Sending"
	case Failure:
		return "Failure"
	default:
		return "Unknown State"
	}
}

func (ss SubState) IsReady() bool {
	is := ss&Ready != 0
	return is
}

func (ss SubState) Pure() SubState {
	noSubs := ^(NACK | ACK)
	state := ss & noSubs
	return state
}

type StateKey struct {
	DID   string
	Nonce string
}

func NewStateKey(agent comm.Receiver, nonce string) StateKey {
	meDID := agent.Trans().MessagePipe().In.Did()
	return StateKey{
		DID:   meDID,
		Nonce: nonce,
	}
}

func (key *StateKey) Data() []byte {
	return []byte(key.DID + key.Nonce)
}

type PayloadInfo struct {
	Type string
}

type State struct {
	Timestamp int64
	T         comm.Task
	PLInfo    PayloadInfo
	Sub       SubState
}

type PSM struct {
	Key    StateKey
	InDID  string
	States []State
}

func NewPSM(d []byte) *PSM {
	p := &PSM{}
	dto.FromGOB(d, p)
	return p
}

func (p *PSM) Data() []byte {
	return dto.ToGOB(p)
}

func (p *PSM) key() []byte {
	return p.Key.Data()
}

func (p *PSM) IsReady() bool {
	if lastState := p.lastState(); lastState != nil {
		return lastState.Sub.IsReady() ||
			lastState.Sub.Pure() == Failure // TODO: until we have recovery for PSM
	}
	return false
}

func (p *PSM) PairwiseName() string {
	defer err2.CatchTrace(func(err error) {
		glog.Error("error in get pw name:", err)
	})

	if state := p.FirstState(); state != nil && state.T.Message != "" {
		return state.T.Message
	}
	if p.InDID != "" {
		r := comm.ActiveRcvrs.Get(p.Key.DID)
		if r == nil {
			return ""
		}
		_, pwName := err2.StrStr.Try(r.FindPW(p.InDID))
		return pwName
	}
	return ""
}

func (p *PSM) Timestamp() int64 {
	if state := p.lastState(); state != nil {
		return state.Timestamp
	}
	return 0
}

func (p *PSM) Next() string {
	if state := p.lastState(); state != nil {
		// todo: when pltype.Termination is the len() returns 0
		if len(state.PLInfo.Type) > 0 {
			return mesg.ProtocolMsgForType(state.PLInfo.Type)
		}
	}
	glog.Warning("no payload type found for PSM!", p.InDID)
	return ""
}

func (p *PSM) PendingUserAction() bool {
	if state := p.lastState(); state != nil {
		// todo: when pltype.Termination is the len() returns 0
		if len(state.PLInfo.Type) > 0 {
			return pltype.UserAction == mesg.ProtocolMsgForType(state.PLInfo.Type)
		}
	}
	glog.Warning("no payload type found for PSM!", p.InDID)
	return false
}

func (p *PSM) FirstState() *State {
	sCount := len(p.States)
	if sCount > 0 {
		return &p.States[0]
	}
	return nil
}

func (p *PSM) lastState() *State {
	sCount := len(p.States)
	if sCount > 0 {
		return &p.States[sCount-1]
	}
	return nil
}

func (p *PSM) Protocol() string {
	if len(p.States) > 0 {
		return mesg.ProtocolForType(p.States[0].PLInfo.Type)
	}
	log.Println("WARNING: no protocol found for state!", p.InDID)
	return ""
}

// TaskFor returns Task of the PSM which corresponds PL.Type. If Type is not
// given it returns last state's Task.
func (p *PSM) TaskFor(plType string) (t *comm.Task) {
	if plType == "" {
		return &p.lastState().T
	}

	for _, s := range p.States {
		if s.PLInfo.Type == plType {
			return &s.T
		}
	}
	return nil
}

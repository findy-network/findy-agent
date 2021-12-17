package psm

import (
	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/pltype"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/golang/glog"
	"github.com/lainio/err2"
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
	ACK --> [*]
	state ACK {
		state "Archiving" as ACK_ARCH

		[*] --> ACK_ARCH
		ACK_ARCH --> [*]
		[*] --> Archived
		Archived --> [*]
	}
	[*] --> NACK
	NACK --> [*]
	state NACK {
		state "Archived" as NACK_ARCHED

		[*] --> Archiving
		Archiving --> [*]
		[*] --> NACK_ARCHED
		NACK_ARCHED --> [*]
	}


}
@enduml
*/

// SubState is enumeration for the state transitions PSM will have during its
// execution. The above PUML diagram illustrates what transitions are currently
// recognized. The Ready state should have 2 internal states: ACK/NACK
type SubState uint // TODO: rename State or StateType

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
	Archiving
	Archived
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
	case Archiving:
		return "Archiving"
	case Archived:
		return "Archived"
	case ReadyACK | Archiving:
		return "ReadyACKArchiving"
	case ReadyACK | Archived:
		return "ReadyACKArchived"
	case ReadyNACK | Archiving:
		return "ReadyNACKArchiving"
	case ReadyNACK | Archived:
		return "ReadyNACKArchived"
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
	return StateKey{
		DID:   agent.MyDID().Did(),
		Nonce: nonce,
	}
}

func (key StateKey) Data() []byte {
	return []byte(key.DID + "|" + key.Nonce)
}

func (key StateKey) String() string {
	return key.DID + "|" + key.Nonce
}

type PayloadInfo struct {
	Type string
}

// todo: new idea StateEvent, rename atleast T
type State struct {
	Timestamp int64
	T         comm.Task
	PLInfo    PayloadInfo
	Sub       SubState
}

// PSM is Protocol State Machine that works in event sourcing principle, i.e.
// every state transition is saved to its State field. Other fields are
// calculated ASAP and kept that way until to the end.
type PSM struct {
	// Key is the primary key of the protocol state machine: it's pointed by
	// CA's DID and the current connection ID
	Key StateKey

	// StartedByUs tells if Our CA is the one who sent the first protocol msg.
	// It' false if we are the receving part. Please note that Role is a
	// protocol specific and not directly correlate with StartedByUs flag.
	StartedByUs bool

	// Role is a protocol role in the current DID protocol
	Role pb.Protocol_Role

	// ConnDID stores our end's pairwise/connection DID. Please note that the
	// connection ID can be found from Key.
	ConnDID string

	// States has all ouf the state history of this PSM in timestamp order
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

func (p *PSM) IsReady() bool {
	if lastState := p.LastState(); lastState != nil {
		return lastState.Sub.IsReady() ||
			lastState.Sub.Pure() == Failure // TODO: until we have recovery for PSM
	}
	return false
}

func (p *PSM) PairwiseName() string {
	defer err2.CatchTrace(func(err error) {
		glog.Error("error in get pw name:", err)
	})

	if state := p.FirstState(); state != nil && state.T.ConnectionID() != "" {
		return state.T.ConnectionID()
	}
	if p.ConnDID != "" {
		r := comm.ActiveRcvrs.Get(p.Key.DID)
		if r == nil {
			return ""
		}
		_, pwName := err2.StrStr.Try(r.FindPWByDID(p.ConnDID))
		return pwName
	}
	return ""
}

func (p *PSM) Timestamp() int64 {
	if state := p.LastState(); state != nil {
		return state.Timestamp
	}
	return 0
}

// Next is for getting the upcoming protocol message type. For example, if we
// are waiting a certain message from other end, we can check the message type
// with this function.
func (p *PSM) Next() string {
	if state := p.LastState(); state != nil {
		// todo: when type is pltype.Termination or Nothing, len() returns 0
		if len(state.PLInfo.Type) > 0 {
			return aries.ProtocolMsgForType(state.PLInfo.Type)
		}
	}
	glog.Warning("no payload type found for PSM!", p.ConnDID)
	return ""
}

// PendingUserAction returns true if we the PSM is waiting an user action msg.
func (p *PSM) PendingUserAction() bool {
	if state := p.LastState(); state != nil {
		// todo: when type is pltype.Termination or Nothing, len() returns 0
		if len(state.PLInfo.Type) > 0 {
			return pltype.UserAction == aries.ProtocolMsgForType(state.PLInfo.Type)
		}
	}
	return false
}

func (p *PSM) FirstState() *State {
	sCount := len(p.States)
	if sCount > 0 {
		return &p.States[0]
	}
	return nil
}

func (p *PSM) LastState() *State {
	sCount := len(p.States)
	if sCount > 0 {
		return &p.States[sCount-1]
	}
	return nil
}

func (p *PSM) Protocol() string {
	if len(p.States) > 0 && p.States[0].PLInfo.Type != "" {
		return aries.ProtocolForType(p.States[0].PLInfo.Type)
	}
	return ""
}

// PresentTask returns latest state's Task of the PSM.
func (p *PSM) PresentTask() (t comm.Task) {
	return p.LastState().T
}

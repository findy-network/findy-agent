package comm

import (
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/service"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
)

func init() {
	gob.Register(&TaskBase{})
}

type Task interface {
	ID() string                     // Unique uid
	Type() string                   // Our internal payload type
	ProtocolType() pb.Protocol_Type // Aries protocol
	UserActionType() string         // Internal payload type when waiting for user action
	Role() pb.Protocol_Role         // Agent role in Aries protocol
	ConnectionID() string           // Pairwise id
	ReceiverEndp() service.Addr     // Pairwise receiver endpoint
	SetReceiverEndp(r service.Addr)
}

type TaskHeader struct {
	TaskID           string
	TypeID           string
	ProtocolRole     pb.Protocol_Role
	ConnID           string
	UserActionPLType string

	Sender   service.Addr
	Receiver service.Addr
}

type TaskBase struct {
	Task
	TaskHeader
}

func (t *TaskBase) ID() string {
	return t.TaskID
}

func (t *TaskBase) Type() string {
	return t.TypeID
}

func (t *TaskBase) ProtocolType() pb.Protocol_Type {
	return pltype.ProtocolTypeForFamily(mesg.ProtocolForType(t.TypeID))
}

func (t *TaskBase) Role() pb.Protocol_Role {
	return t.ProtocolRole
}

func (t *TaskBase) ConnectionID() string {
	return t.ConnID
}

func (t *TaskBase) UserActionType() string {
	return t.UserActionPLType
}

func (t *TaskBase) ReceiverEndp() service.Addr {
	return t.Receiver
}

func (t *TaskBase) SetReceiverEndp(r service.Addr) {
	t.Receiver = r
}

// SwitchDirection changes SenderEndp and ReceiverEndp data
func (t *TaskHeader) SwitchDirection() {
	tmp := t.Sender
	t.Sender = t.Receiver
	t.Receiver = tmp
}

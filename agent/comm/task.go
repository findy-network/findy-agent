package comm

import (
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/service"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
)

func init() {
	gob.Register(&TaskBase{})
}

type Task interface {
	ID() string
	Type() string
	UserActionType() string
	Role() pb.Protocol_Role
	ConnectionID() string
	ReceiverEndp() service.Addr
	SetReceiverEndp(r service.Addr)
}

type TaskHeader struct {
	TaskID         string
	TypeID         string
	ProtocolTypeID string
	ProtocolRole   pb.Protocol_Role
	ConnID         string
	UAType         string

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

func (t *TaskBase) Role() pb.Protocol_Role {
	return t.ProtocolRole
}

func (t *TaskBase) ConnectionID() string {
	return t.ConnID
}

func (t *TaskBase) UserActionType() string {
	return t.UAType
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

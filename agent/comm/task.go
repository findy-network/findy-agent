package comm

import (
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/service"
)

func init() {
	gob.Register(&TaskBase{})
}

type Task interface {
	ID() string
	Type() string
	Role() string
	ConnectionID() string
	ReceiverEndp() service.Addr
	SetReceiverEndp(r service.Addr)
}

type TaskHeader struct {
	ID             string
	TypeID         string
	ProtocolTypeID string
	Role           string
	PrevThreadID   string
	ConnectionID   string

	Sender   service.Addr
	Receiver service.Addr
}

type TaskBase struct {
	Task
	Head TaskHeader
}

func (t *TaskBase) ID() string {
	return t.Head.ID
}

func (t *TaskBase) Type() string {
	return t.Head.TypeID
}

func (t *TaskBase) Role() string {
	return t.Head.Role
}

func (t *TaskBase) ConnectionID() string {
	return t.Head.ConnectionID
}

func (t *TaskBase) ReceiverEndp() service.Addr {
	return t.Head.Receiver
}

func (t *TaskBase) SetReceiverEndp(r service.Addr) {
	t.Head.Receiver = r
}

// SwitchDirection changes SenderEndp and ReceiverEndp data
func (t *TaskHeader) SwitchDirection() {
	tmp := t.Sender
	t.Sender = t.Receiver
	t.Receiver = tmp
}

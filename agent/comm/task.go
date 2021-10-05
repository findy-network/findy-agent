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
	ID() string
	Type() string
	ProtocolType() pb.Protocol_Type
	UserActionType() string
	SetUserActionType(userActionType string)
	Role() pb.Protocol_Role
	ConnectionID() string
	ReceiverEndp() service.Addr
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
	protocol := mesg.ProtocolForType(t.TypeID)
	switch protocol {
	case pltype.ProtocolBasicMessage:
		return pb.Protocol_BASIC_MESSAGE
	case pltype.ProtocolConnection:
		return pb.Protocol_DIDEXCHANGE
	case pltype.ProtocolIssueCredential:
		return pb.Protocol_ISSUE_CREDENTIAL
	case pltype.ProtocolPresentProof:
		return pb.Protocol_PRESENT_PROOF
	case pltype.ProtocolTrustPing:
		return pb.Protocol_TRUST_PING
	}
	return pb.Protocol_NONE
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

func (t *TaskBase) SetUserActionType(userActionType string) {
	t.UserActionPLType = userActionType
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

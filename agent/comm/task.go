package comm

import (
	"errors"

	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/std/didexchange/invitation"
	aries "github.com/findy-network/findy-agent/std/didexchange/invitation"
	"github.com/findy-network/findy-common-go/dto"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/golang/glog"
)

type TaskHeader struct {
	ID           string
	TypeID       string
	Role         string
	PrevThreadID string
	ConnectionID string

	SenderEndp   service.Addr
	ReceiverEndp service.Addr
}

type TaskDIDExchange struct {
	Invitation   *aries.Invitation
	InvitationID string
	Label        string
}

type Task struct {
	Nonce   string
	TypeID  string // "connection", "issue-credential", "trust_ping"
	Message string
	Info    string

	// Pairwise
	ConnectionInvitation *invitation.Invitation

	// Issue credential
	CredentialAttrs *[]didcomm.CredentialAttribute
	CredDefID       *string

	// Present proof
	ProofAttrs      *[]didcomm.ProofAttribute
	ProofPredicates *[]didcomm.ProofPredicate

	header      *TaskHeader
	didExchange *TaskDIDExchange
}

func CreateStartTask(t *Task, protocol *pb.Protocol) *Task {
	t.header = &TaskHeader{
		TypeID: t.TypeID,
		//TypeID:       protocol.GetTypeID().String(),
		Role:         protocol.GetRole().String(),
		PrevThreadID: protocol.GetPrevThreadID(),
		ConnectionID: protocol.GetConnectionID(),
	}
	switch protocol.TypeID {
	case pb.Protocol_DIDEXCHANGE:
		t.didExchange = &TaskDIDExchange{}
		if protocol.GetDIDExchange() == nil {
			panic(errors.New("connection attrs cannot be nil"))
		}
		var invitation aries.Invitation
		dto.FromJSONStr(protocol.GetDIDExchange().GetInvitationJSON(), &invitation)
		t.didExchange.Invitation = &invitation
		t.didExchange.Label = protocol.GetDIDExchange().GetLabel()
		t.didExchange.InvitationID = invitation.ID
		glog.V(1).Infof("Create task for DIDExchange with invitation id %s", invitation.ID)
	}

	return t
}

func CreateTask(typeID, id, connectionID string, receiver, sender *service.Addr) *Task {
	var r, s service.Addr
	if receiver != nil {
		r = *receiver
	}
	if sender != nil {
		s = *sender
	}
	t := &Task{
		Nonce:  id,
		TypeID: typeID,
		header: &TaskHeader{
			ID:           id,
			TypeID:       typeID,
			ReceiverEndp: r,
			SenderEndp:   s,
			ConnectionID: connectionID,
		},
		Message: connectionID,
	}
	return t
}

func CreateTaskWithData(t *Task, receiver, sender *service.Addr) *Task {
	var r, s service.Addr
	if receiver != nil {
		r = *receiver
	}
	if sender != nil {
		s = *sender
	}
	t.header = &TaskHeader{
		ReceiverEndp: r,
		SenderEndp:   s,
	}
	return t
}

func (t *Task) SetReceiver(endpoint *service.Addr) {
	t.header.ReceiverEndp = *endpoint
}

func (t *Task) GetHeader() *TaskHeader {
	return t.header
}

// SwitchDirection changes SenderEndp and ReceiverEndp data
func (t *TaskHeader) SwitchDirection() {
	tmp := t.SenderEndp
	t.SenderEndp = t.ReceiverEndp
	t.ReceiverEndp = tmp
}

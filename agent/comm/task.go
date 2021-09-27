package comm

import (
	"errors"

	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/service"
	aries "github.com/findy-network/findy-agent/std/didexchange/invitation"
	"github.com/findy-network/findy-common-go/dto"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/golang/glog"
	"github.com/lainio/err2/assert"
)

type TaskHeader struct {
	ID             string
	TypeID         string
	ProtocolTypeID string
	Role           string
	PrevThreadID   string
	ConnectionID   string

	SenderEndp   service.Addr
	ReceiverEndp service.Addr
}

type TaskDIDExchange struct {
	Invitation   *aries.Invitation
	InvitationID string
	Label        string
}

type TaskBasicMessage struct {
	Content string
}

type TaskIssueCredential struct {
	Comment         string
	CredentialAttrs []didcomm.CredentialAttribute
	CredDefID       string
}

type TaskPresentProof struct {
	Comment         string
	ProofAttrs      []didcomm.ProofAttribute
	ProofPredicates []didcomm.ProofPredicate
}

type Task struct {
	Header          *TaskHeader
	DidExchange     *TaskDIDExchange
	BasicMessage    *TaskBasicMessage
	IssueCredential *TaskIssueCredential
	PresentProof    *TaskPresentProof
}

func CreateStartTask(typeID, id string, protocol *pb.Protocol) *Task {
	t := &Task{
		Header: &TaskHeader{
			ID:             id,
			TypeID:         typeID,
			ProtocolTypeID: protocol.GetTypeID().String(),
			Role:           protocol.GetRole().String(),
			PrevThreadID:   protocol.GetPrevThreadID(),
			ConnectionID:   protocol.GetConnectionID(),
		},
	}
	switch protocol.TypeID {
	case pb.Protocol_DIDEXCHANGE:
		if protocol.GetDIDExchange() == nil {
			panic(errors.New("connection attrs cannot be nil"))
		}
		var invitation aries.Invitation
		dto.FromJSONStr(protocol.GetDIDExchange().GetInvitationJSON(), &invitation)
		t.DidExchange = &TaskDIDExchange{
			Invitation:   &invitation,
			Label:        protocol.GetDIDExchange().GetLabel(),
			InvitationID: invitation.ID,
		}
		t.Header.ID = invitation.ID
		glog.V(1).Infof("Create task for DIDExchange with invitation id %s", invitation.ID)
	case pb.Protocol_BASIC_MESSAGE:
		t.BasicMessage = &TaskBasicMessage{
			Content: protocol.GetBasicMessage().Content,
		}
		glog.V(1).Infof("Create task for BasicMessage with connection id %s", t.Header.ConnectionID)

	case pb.Protocol_ISSUE_CREDENTIAL:
		t.IssueCredential = &TaskIssueCredential{}
		cred := protocol.GetIssueCredential()
		if cred == nil || (protocol.GetRole() != pb.Protocol_INITIATOR && protocol.GetRole() != pb.Protocol_ADDRESSEE) {
			panic(errors.New("cred cannot be nil for issuing protocol, role is needed"))
		}

		var credAttrs []didcomm.CredentialAttribute
		if cred.GetAttributesJSON() != "" {
			dto.FromJSONStr(cred.GetAttributesJSON(), &credAttrs)
			glog.V(3).Infoln("set cred attrs from json")
		} else {
			assert.P.True(cred.GetAttributes() != nil)
			credAttrs = make([]didcomm.CredentialAttribute, len(cred.GetAttributes().GetAttributes()))
			for i, attribute := range cred.GetAttributes().GetAttributes() {
				credAttrs[i] = didcomm.CredentialAttribute{
					Name:  attribute.Name,
					Value: attribute.Value,
				}
			}
			glog.V(3).Infoln("set cred from attrs")
		}
		t.IssueCredential.CredentialAttrs = credAttrs
		t.IssueCredential.CredDefID = cred.CredDefID
		glog.V(1).Infof(
			"Create task for IssueCredential with connection id %s, role %s",
			t.Header.ConnectionID,
			protocol.GetRole().String(),
		)
	case pb.Protocol_PRESENT_PROOF:
		t.PresentProof = &TaskPresentProof{}
		proof := protocol.GetPresentProof()
		if proof == nil || (protocol.GetRole() != pb.Protocol_INITIATOR && protocol.GetRole() != pb.Protocol_ADDRESSEE) {
			panic(errors.New("proof cannot be nil for present proof protocol, role is needed"))
		}

		// attributes - mandatory
		var proofAttrs []didcomm.ProofAttribute
		if proof.GetAttributesJSON() != "" {
			dto.FromJSONStr(proof.GetAttributesJSON(), &proofAttrs)
			glog.V(3).Infoln("set proof attrs from json:", proof.GetAttributesJSON())
		} else {
			assert.P.True(proof.GetAttributes() != nil)
			proofAttrs = make([]didcomm.ProofAttribute, len(proof.GetAttributes().GetAttributes()))
			for i, attribute := range proof.GetAttributes().GetAttributes() {
				proofAttrs[i] = didcomm.ProofAttribute{
					ID:        attribute.ID,
					Name:      attribute.Name,
					CredDefID: attribute.CredDefID,
				}
			}
			glog.V(3).Infoln("set proof from attrs")
		}

		// predicates - optional
		var proofPredicates []didcomm.ProofPredicate
		if proof.GetPredicatesJSON() != "" {
			dto.FromJSONStr(proof.GetPredicatesJSON(), &proofPredicates)
			glog.V(3).Infoln("set proof predicates from json:", proof.GetPredicatesJSON())
		} else if proof.GetPredicates() != nil {
			proofPredicates = make([]didcomm.ProofPredicate, len(proof.GetPredicates().GetPredicates()))
			for i, predicate := range proof.GetPredicates().GetPredicates() {
				proofPredicates[i] = didcomm.ProofPredicate{
					ID:     predicate.ID,
					Name:   predicate.Name,
					PType:  predicate.PType,
					PValue: predicate.PValue,
				}
			}
			glog.V(3).Infoln("set proof from predicates")
		}

		t.PresentProof.ProofAttrs = proofAttrs
		t.PresentProof.ProofPredicates = proofPredicates

		glog.V(1).Infof(
			"Create task for PresentProof with connection id %s, role %s",
			t.Header.ConnectionID,
			protocol.GetRole().String(),
		)

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
		Header: &TaskHeader{
			ID:           id,
			TypeID:       typeID,
			ReceiverEndp: r,
			SenderEndp:   s,
			ConnectionID: connectionID,
		},
	}
	return t
}

func (t *Task) SetReceiver(endpoint *service.Addr) {
	t.Header.ReceiverEndp = *endpoint
}

func (t *Task) GetHeader() *TaskHeader {
	return t.Header
}

func (t *Task) GetDIDExchange() *TaskDIDExchange {
	return t.DidExchange
}

func (t *Task) GetBasicMessage() *TaskBasicMessage {
	return t.BasicMessage
}

func (t *Task) GetPresentProof() *TaskPresentProof {
	return t.PresentProof
}

func (t *Task) GetIssueCredential() *TaskIssueCredential {
	return t.IssueCredential
}

// SwitchDirection changes SenderEndp and ReceiverEndp data
func (t *TaskHeader) SwitchDirection() {
	tmp := t.SenderEndp
	t.SenderEndp = t.ReceiverEndp
	t.ReceiverEndp = tmp
}

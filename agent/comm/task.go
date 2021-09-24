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
	Comment string
}

type TaskPresentProof struct {
	Comment string
}

type Task struct {
	Nonce string

	// Pairwise
	ConnectionInvitation *invitation.Invitation

	// Issue credential
	CredentialAttrs *[]didcomm.CredentialAttribute
	CredDefID       *string

	// Present proof
	ProofAttrs      *[]didcomm.ProofAttribute
	ProofPredicates *[]didcomm.ProofPredicate

	Header          *TaskHeader
	DidExchange     *TaskDIDExchange
	BasicMessage    *TaskBasicMessage
	IssueCredential *TaskIssueCredential
	PresentProof    *TaskPresentProof
}

func CreateStartTask(typeID string, t *Task, protocol *pb.Protocol) *Task {
	t.Header = &TaskHeader{
		ID:             t.Nonce,
		TypeID:         typeID,
		ProtocolTypeID: protocol.GetTypeID().String(),
		Role:           protocol.GetRole().String(),
		PrevThreadID:   protocol.GetPrevThreadID(),
		ConnectionID:   protocol.GetConnectionID(),
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
		glog.V(1).Infof("Create task for DIDExchange with invitation id %s", invitation.ID)
	case pb.Protocol_BASIC_MESSAGE:
		t.BasicMessage = &TaskBasicMessage{
			Content: protocol.GetBasicMessage().Content,
		}
		glog.V(1).Infof("Create task for BasicMessage with connection id %s", t.Header.ConnectionID)

	case pb.Protocol_ISSUE_CREDENTIAL:
		t.IssueCredential = &TaskIssueCredential{}

		/*if protocol.Role == pb.Protocol_INITIATOR || protocol.Role == pb.Protocol_ADDRESSEE {
			credDef := protocol.GetIssueCredential()
			if credDef == nil {
				panic(errors.New("cred def cannot be nil for issuing protocol"))
			}
			task.CredDefID = &credDef.CredDefID

			if credDef.GetAttributes() != nil {
				attributes := make([]didcomm.CredentialAttribute, len(credDef.GetAttributes().GetAttributes()))
				for i, attribute := range credDef.GetAttributes().GetAttributes() {
					attributes[i] = didcomm.CredentialAttribute{
						Name:  attribute.Name,
						Value: attribute.Value,
					}
				}
				task.CredentialAttrs = &attributes
				glog.V(1).Infoln("set cred from attrs")
			} else if credDef.GetAttributesJSON() != "" {
				var credAttrs []didcomm.CredentialAttribute
				dto.FromJSONStr(credDef.GetAttributesJSON(), &credAttrs)
				task.CredentialAttrs = &credAttrs
				glog.V(1).Infoln("set cred attrs from json")
			}
		}*/
	case pb.Protocol_PRESENT_PROOF:
		t.PresentProof = &TaskPresentProof{}
		/*if protocol.Role == pb.Protocol_INITIATOR || protocol.Role == pb.Protocol_ADDRESSEE {
			proofReq := protocol.GetPresentProof()

			// Attributes
			if proofReq.GetAttributesJSON() != "" {
				var proofAttrs []didcomm.ProofAttribute
				dto.FromJSONStr(proofReq.GetAttributesJSON(), &proofAttrs)
				task.ProofAttrs = &proofAttrs
				glog.V(1).Infoln("set proof attrs from json:", proofReq.GetAttributesJSON())
			} else if proofReq.GetAttributes() != nil {
				attributes := make([]didcomm.ProofAttribute, len(proofReq.GetAttributes().GetAttributes()))
				for i, attribute := range proofReq.GetAttributes().GetAttributes() {
					attributes[i] = didcomm.ProofAttribute{
						ID:        attribute.ID,
						Name:      attribute.Name,
						CredDefID: attribute.CredDefID,
						//Predicate: attribute.Predicate,
					}
				}
				task.ProofAttrs = &attributes
				glog.V(1).Infoln("set proof from attrs")
			}

			// Predicates
			if proofReq.GetPredicatesJSON() != "" {
				var proofPredicates []didcomm.ProofPredicate
				dto.FromJSONStr(proofReq.GetPredicatesJSON(), &proofPredicates)
				task.ProofPredicates = &proofPredicates
				glog.V(1).Infoln("set proof predicates from json:", proofReq.GetPredicatesJSON())
			} else if proofReq.GetPredicates() != nil {
				predicates := make([]didcomm.ProofPredicate, len(proofReq.GetPredicates().GetPredicates()))
				for i, predicate := range proofReq.GetPredicates().GetPredicates() {
					predicates[i] = didcomm.ProofPredicate{
						ID:     predicate.ID,
						Name:   predicate.Name,
						PType:  predicate.PType,
						PValue: predicate.PValue,
					}
				}
				task.ProofPredicates = &predicates
				glog.V(1).Infoln("set proof from predicates")
			}
		}*/
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
		Nonce: id,
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

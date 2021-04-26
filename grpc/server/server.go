/*
Package server is a playground where we have tried gRPC interface for CA API.
*/
package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/utils"
	didexchange "github.com/findy-network/findy-agent/std/didexchange/invitation"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	ops "github.com/findy-network/findy-common-go/grpc/ops/v1"
	"github.com/findy-network/findy-common-go/jwt"
	"github.com/findy-network/findy-common-go/rpc"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"google.golang.org/grpc"
)

var Server *grpc.Server

func Serve(conf *rpc.ServerCfg) {
	if conf == nil {
		panic("GRPC server needs configuration")
	}
	if conf.PKI != nil {
		glog.V(1).Infof("starting gRPC server with\ncrt:\t%s\nkey:\t%s\nclient:\t%s",
			conf.PKI.Server.CertFile, conf.PKI.Server.KeyFile, conf.PKI.Client.CertFile)
	}

	conf.Register = func(s *grpc.Server) error {
		pb.RegisterProtocolServiceServer(s, &didCommServer{})
		pb.RegisterAgentServiceServer(s, &agentServer{})

		root := utils.Settings.GRPCAdmin()
		ops.RegisterAgencyServiceServer(s, &agencyService{Root: root})
		ops.RegisterDevOpsServiceServer(s, &devOpsServer{Root: root})

		glog.V(3).Infoln("GRPC OK")
		return nil
	}

	s, lis, err := rpc.PrepareServe(conf)
	err2.Check(err)
	Server = s
	err2.Check(s.Serve(lis))
}

func taskFrom(protocol *pb.Protocol) (t *comm.Task, err error) {
	defer err2.Return(&err)

	task := &comm.Task{
		Nonce:   utils.UUID(),
		TypeID:  uniqueTypeID(protocol.Role, protocol.TypeID),
		Message: protocol.ConnectionID,
	}
	switch protocol.TypeID {
	case pb.Protocol_TRUST_PING:
		if protocol.ConnectionID == "" {
			glog.Warningln("pinging first found connection, conn-id was empty")
		}
	case pb.Protocol_BASIC_MESSAGE:
		task.Info = protocol.GetBasicMessage().Content
		glog.V(1).Infoln("basic_message content:", task.Info)
	case pb.Protocol_DIDEXCHANGE:
		if protocol.GetDIDExchange() == nil {
			panic(errors.New("connection attrs cannot be nil"))
		}
		var invitation didexchange.Invitation
		dto.FromJSONStr(protocol.GetDIDExchange().GetInvitationJson(), &invitation)
		task.ConnectionInvitation = &invitation
		task.Info = protocol.GetDIDExchange().GetLabel()
		task.Nonce = invitation.ID // Important!! we must use same id!
		glog.V(1).Infoln("set invitation")
	case pb.Protocol_ISSUE_CREDENTIAL:
		if protocol.Role == pb.Protocol_INITIATOR {
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
			} else if credDef.GetAttributesJson() != "" {
				var credAttrs []didcomm.CredentialAttribute
				dto.FromJSONStr(credDef.GetAttributesJson(), &credAttrs)
				task.CredentialAttrs = &credAttrs
				glog.V(1).Infoln("set cred attrs from json")
			}
		}
	case pb.Protocol_PRESENT_PROOF:
		if protocol.Role == pb.Protocol_INITIATOR {
			proofReq := protocol.GetPresentProof()
			if proofReq.GetAttributesJson() != "" {
				var proofAttrs []didcomm.ProofAttribute
				dto.FromJSONStr(proofReq.GetAttributesJson(), &proofAttrs)
				task.ProofAttrs = &proofAttrs
				glog.V(1).Infoln("set proof attrs from json:", proofReq.GetAttributesJson())
			} else if proofReq.GetAttributes() != nil {
				attributes := make([]didcomm.ProofAttribute, len(proofReq.GetAttributes().GetAttributes()))
				for i, attribute := range proofReq.GetAttributes().GetAttributes() {
					attributes[i] = didcomm.ProofAttribute{
						Name:      attribute.Name,
						CredDefID: attribute.CredDefID,
						//Predicate: attribute.Predicate,
					}
				}
				task.ProofAttrs = &attributes
				glog.V(1).Infoln("set proof from attrs")
			}
		}
	}
	return task, nil
}

var notificationTypeID = map[string]pb.Notification_Type{
	pltype.CANotifyStatus:     pb.Notification_STATUS_UPDATE,
	pltype.CANotifyUserAction: pb.Notification_ACTION_NEEDED,
}

var questionTypeID = map[string]pb.Question_Type{
	pltype.SAPing:                         pb.Question_ANSWER_NEEDED_PING,
	pltype.SAIssueCredentialAcceptPropose: pb.Question_ANSWER_NEEDED_ISSUE_PROPOSE,
	pltype.SAPresentProofAcceptPropose:    pb.Question_ANSWER_NEEDED_PROOF_PROPOSE,
	pltype.SAPresentProofAcceptValues:     pb.Question_ANSWER_NEEDED_PROOF_VERIFY,
}

func uniqueTypeID(role pb.Protocol_Role, id pb.Protocol_Type) string {
	i := int32(10*role) + int32(id)
	glog.V(5).Infoln("unique id:", i, typeID[i])
	s, ok := typeID[i]
	if !ok {
		msg := fmt.Sprintf("cannot find typeid for (%d+%d)", role, id)
		glog.Error(msg)
		panic(msg)
	}
	return s
}

// typeID is look up table for
var typeID = map[int32]string{
	int32(10*pb.Protocol_INITIATOR) + int32(pb.Protocol_DIDEXCHANGE):      pltype.CAPairwiseCreate,
	int32(10*pb.Protocol_INITIATOR) + int32(pb.Protocol_ISSUE_CREDENTIAL): pltype.CACredOffer,
	int32(10*pb.Protocol_ADDRESSEE) + int32(pb.Protocol_ISSUE_CREDENTIAL): pltype.CACredRequest,
	int32(10*pb.Protocol_INITIATOR) + int32(pb.Protocol_PRESENT_PROOF):    pltype.CAProofRequest,
	int32(10*pb.Protocol_ADDRESSEE) + int32(pb.Protocol_PRESENT_PROOF):    pltype.CAProofPropose,
	int32(10*pb.Protocol_INITIATOR) + int32(pb.Protocol_TRUST_PING):       pltype.CATrustPing,
	int32(10*pb.Protocol_INITIATOR) + int32(pb.Protocol_BASIC_MESSAGE):    pltype.CABasicMessage,
	int32(10*pb.Protocol_RESUMER) + int32(pb.Protocol_ISSUE_CREDENTIAL):   pltype.CAContinueIssueCredentialProtocol,
	int32(10*pb.Protocol_RESUMER) + int32(pb.Protocol_PRESENT_PROOF):      pltype.CAContinuePresentProofProtocol,
}

// to get protocol family
var protocolName = [...]string{
	pltype.Nothing,                 // NONE
	pltype.AriesProtocolConnection, // "CONNECT",
	pltype.ProtocolIssueCredential, // "ISSUE",
	pltype.ProtocolPresentProof,    // "PROOF",
	pltype.ProtocolTrustPing,       // "TRUST_PING",
	pltype.ProtocolBasicMessage,    // "BASIC_MESSAGE",
}

var protocolType = map[string]pb.Protocol_Type{
	pltype.AriesProtocolConnection: pb.Protocol_DIDEXCHANGE,
	pltype.ProtocolIssueCredential: pb.Protocol_ISSUE_CREDENTIAL,
	pltype.ProtocolPresentProof:    pb.Protocol_PRESENT_PROOF,
	pltype.ProtocolTrustPing:       pb.Protocol_TRUST_PING,
	pltype.ProtocolBasicMessage:    pb.Protocol_BASIC_MESSAGE,
}

var roleType = map[bool]pb.Protocol_Role{
	true:  pb.Protocol_INITIATOR, // is Initiator
	false: pb.Protocol_ADDRESSEE, //
}

func ca(ctx context.Context) (caDID string, r comm.Receiver, err error) {
	caDID = jwt.User(ctx)
	if !agency.IsHandlerInThisAgency(caDID) {
		return "", nil, fmt.Errorf("handler (%s) is not in this agency", caDID)
	}
	rcvr, ok := agency.Handler(caDID).(comm.Receiver)
	if !ok {
		return "", nil, fmt.Errorf("no ca did (%s)", caDID)
	}
	glog.V(1).Infoln("grpc call with caDID:", caDID)
	return caDID, rcvr, nil
}

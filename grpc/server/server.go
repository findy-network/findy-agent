/*
Package server is a playground where we have tried gRPC interface for CA API.
*/
package server

import (
	"context"
	"errors"

	pb "github.com/findy-network/findy-agent-api/grpc/agency"
	"github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/utils"
	didexchange "github.com/findy-network/findy-agent/std/didexchange/invitation"
	"github.com/findy-network/findy-grpc/jwt"
	"github.com/findy-network/findy-grpc/rpc"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"google.golang.org/grpc"
)

func Serve() {
	pki := rpc.LoadPKI()
	glog.V(1).Infof("starting gRPC server with\ncrt:\t%s\nkey:\t%s\nclient:\t%s",
		pki.Server.CertFile, pki.Server.KeyFile, pki.Client.CertFile)

	rpc.Serve(rpc.ServerCfg{
		Port: 50051,
		TLS:  true,
		PKI:  *pki,
		Register: func(s *grpc.Server) error {
			pb.RegisterDIDCommServer(s, &didCommServer{})
			pb.RegisterAgentServer(s, &agentServer{})
			//pb.RegisterAgencyServer(s, &agencyService{})
			pb.RegisterDevOpsServer(s, &devOpsServer{Root: "findy-root"})
			glog.Infoln("GRPC registration IIIIIII OK")
			return nil
		},
	})
}

func taskFrom(protocol *pb.Protocol) (t *comm.Task, err error) {
	defer err2.Return(&err)

	task := &comm.Task{
		Nonce:   utils.UUID(),
		TypeID:  typeID[protocol.TypeId],
		Message: protocol.ConnectionId,
	}
	switch protocol.TypeId {
	case pb.Protocol_CONNECT:
		var invitation didexchange.Invitation
		dto.FromJSONStr(protocol.GetInvitationJson(), &invitation)
		task.ConnectionInvitation = &invitation
		task.Nonce = invitation.ID // Important!! we must use same id!
		glog.V(1).Infoln("set invitation")
	case pb.Protocol_ISSUE, pb.Protocol_PROPOSE_ISSUING:
		credDef := protocol.GetCredDef()
		if credDef == nil {
			panic(errors.New("cred def cannot be nil for issuing protocol"))
		}
		task.CredDefID = &credDef.CredDefId
		var credAttrs []didcomm.CredentialAttribute
		dto.FromJSONStr(credDef.GetAttributesJson(), &credAttrs)
		task.CredentialAttrs = &credAttrs
		glog.V(1).Infoln("set cred attrs")
	case pb.Protocol_REQUEST_PROOF:
		var proofAttrs []didcomm.ProofAttribute
		dto.FromJSONStr(protocol.GetProofAttributesJson(), &proofAttrs)
		task.ProofAttrs = &proofAttrs
		glog.V(1).Infoln("set proof attrs")
	}
	return task, nil
}

var notificationTypeID = map[string]pb.Notification_Type{
	pltype.CANotifyStatus:                 pb.Notification_STATUS_UPDATE,
	pltype.CANotifyUserAction:             pb.Notification_ACTION_NEEDED,
	pltype.SAPing:                         pb.Notification_ACTION_NEEDED_PING,
	pltype.SAIssueCredentialAcceptPropose: pb.Notification_ACTION_NEEDED,
	pltype.SAPresentProofAcceptPropose:    pb.Notification_ACTION_NEEDED,
	pltype.SAPresentProofAcceptValues:     pb.Notification_ACTION_NEEDED,
}

// typeID is look up table for
var typeID = [...]string{
	pltype.CAPairwiseCreate,                  // "CONNECT",
	pltype.CACredOffer,                       // "ISSUE",
	pltype.CACredRequest,                     // "PROPOSE_ISSUING",
	pltype.CAProofRequest,                    // "REQUEST_PROOF",
	pltype.CAProofPropose,                    // "PROPOSE_PROOFING",
	pltype.CATrustPing,                       // "TRUST_PING",
	pltype.CABasicMessage,                    // "BASIC_MESSAGE",
	pltype.CAContinueIssueCredentialProtocol, // "CONTINUE_ISSUE"
	pltype.CAContinuePresentProofProtocol,    // "CONTINUE_PROOF"
}

// typeID is look up table for
var protocolName = [...]string{
	pltype.AriesProtocolConnection, // "CONNECT",
	pltype.ProtocolIssueCredential, // "ISSUE",
	pltype.ProtocolIssueCredential, // "PROPOSE_ISSUING",
	pltype.ProtocolPresentProof,    // "REQUEST_PROOF",
	pltype.ProtocolPresentProof,    // "PROPOSE_PROOFING",
	pltype.ProtocolTrustPing,       // "TRUST_PING",
	pltype.ProtocolBasicMessage,    // "BASIC_MESSAGE",
	pltype.ProtocolIssueCredential, // "CONTINUE_ISSUE"
	pltype.ProtocolPresentProof,    // "CONTINUE_PROOF"
}

func ca(ctx context.Context) (caDID string, r comm.Receiver, err error) {
	caDID = jwt.User(ctx)
	if !agency.IsHandlerInThisAgency(caDID) {
		return "", nil, errors.New("handler is not in this agency")
	}
	rcvr, ok := agency.Handler(caDID).(comm.Receiver)
	if !ok {
		return "", nil, errors.New("no ca did")
	}
	glog.V(3).Infoln("caDID:", caDID)
	return caDID, rcvr, nil
}

/*
Package server is a playground where we have tried gRPC interface for CA API.
*/
package server

import (
	"context"
	"errors"
	"fmt"

	pb "github.com/findy-network/findy-agent-api/grpc/agency"
	ops "github.com/findy-network/findy-agent-api/grpc/ops"
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
	"google.golang.org/grpc/test/bufconn"
)

var Server *grpc.Server

func Serve(testLis *bufconn.Listener) {
	pki := rpc.LoadPKI()
	glog.V(1).Infof("starting gRPC server with\ncrt:\t%s\nkey:\t%s\nclient:\t%s",
		pki.Server.CertFile, pki.Server.KeyFile, pki.Client.CertFile)

	s, lis, err := rpc.PrepareServe(rpc.ServerCfg{
		Port:    50051,
		TLS:     true,
		PKI:     *pki,
		TestLis: testLis,
		Register: func(s *grpc.Server) error {
			pb.RegisterDIDCommServer(s, &didCommServer{})
			pb.RegisterAgentServer(s, &agentServer{})
			//pb.RegisterAgencyServer(s, &agencyService{})
			ops.RegisterDevOpsServer(s, &devOpsServer{Root: "findy-root"})
			glog.Infoln("GRPC registration IIIIIII OK")
			return nil
		},
	})
	err2.Check(err)
	Server = s
	err2.Check(s.Serve(lis))
}

func taskFrom(protocol *pb.Protocol) (t *comm.Task, err error) {
	defer err2.Return(&err)

	task := &comm.Task{
		Nonce:   utils.UUID(),
		TypeID:  uniqueTypeID(protocol.Role, protocol.TypeId),
		Message: protocol.ConnectionId,
	}
	switch protocol.TypeId {
	case pb.Protocol_CONNECT:
		var invitation didexchange.Invitation
		dto.FromJSONStr(protocol.GetInvitationJson(), &invitation)
		task.ConnectionInvitation = &invitation
		task.Nonce = invitation.ID // Important!! we must use same id!
		glog.V(1).Infoln("set invitation")
	case pb.Protocol_ISSUE:
		if protocol.Role == pb.Protocol_INITIATOR {
			credDef := protocol.GetCredDef()
			if credDef == nil {
				panic(errors.New("cred def cannot be nil for issuing protocol"))
			}
			task.CredDefID = &credDef.CredDefId

			// todo: this is first version using json attrs, write impl for 2nd 'one of'
			var credAttrs []didcomm.CredentialAttribute
			dto.FromJSONStr(credDef.GetAttributesJson(), &credAttrs)
			task.CredentialAttrs = &credAttrs

			glog.V(1).Infoln("set cred attrs")
		}
	case pb.Protocol_PROOF:
		if protocol.Role == pb.Protocol_INITIATOR {
			var proofAttrs []didcomm.ProofAttribute
			dto.FromJSONStr(protocol.GetProofAttributesJson(), &proofAttrs)
			task.ProofAttrs = &proofAttrs
			glog.V(1).Infoln("set proof attrs")
		}
	}
	return task, nil
}

var notificationTypeID = map[string]pb.Notification_Type{
	pltype.CANotifyStatus:                 pb.Notification_STATUS_UPDATE,
	pltype.CANotifyUserAction:             pb.Notification_ACTION_NEEDED,
	pltype.SAPing:                         pb.Notification_ANSWER_NEEDED_PING,
	pltype.SAIssueCredentialAcceptPropose: pb.Notification_ANSWER_NEEDED_ISSUE_PROPOSE,
	pltype.SAPresentProofAcceptPropose:    pb.Notification_ANSWER_NEEDED_PROOF_PROPOSE,
	pltype.SAPresentProofAcceptValues:     pb.Notification_ANSWER_NEEDED_PROOF_VERIFY,
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
	int32(10*pb.Protocol_INITIATOR) + int32(pb.Protocol_CONNECT):       pltype.CAPairwiseCreate,
	int32(10*pb.Protocol_INITIATOR) + int32(pb.Protocol_ISSUE):         pltype.CACredOffer,
	int32(10*pb.Protocol_ADDRESSEE) + int32(pb.Protocol_ISSUE):         pltype.CACredRequest,
	int32(10*pb.Protocol_INITIATOR) + int32(pb.Protocol_PROOF):         pltype.CAProofRequest,
	int32(10*pb.Protocol_ADDRESSEE) + int32(pb.Protocol_PROOF):         pltype.CAProofPropose,
	int32(10*pb.Protocol_INITIATOR) + int32(pb.Protocol_TRUST_PING):    pltype.CATrustPing,
	int32(10*pb.Protocol_INITIATOR) + int32(pb.Protocol_BASIC_MESSAGE): pltype.CABasicMessage,
	int32(10*pb.Protocol_RESUME) + int32(pb.Protocol_ISSUE):            pltype.CAContinueIssueCredentialProtocol,
	int32(10*pb.Protocol_RESUME) + int32(pb.Protocol_PROOF):            pltype.CAContinuePresentProofProtocol,
}

// typeID is look up table for
var protocolName = [...]string{
	pltype.AriesProtocolConnection, // "CONNECT",
	pltype.ProtocolIssueCredential, // "ISSUE",
	pltype.ProtocolPresentProof,    // "PROOF",
	pltype.ProtocolTrustPing,       // "TRUST_PING",
	pltype.ProtocolBasicMessage,    // "BASIC_MESSAGE",
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

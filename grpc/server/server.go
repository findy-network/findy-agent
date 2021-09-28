/*
Package server is a playground where we have tried gRPC interface for CA API.
*/
package server

import (
	"context"
	"fmt"

	"github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/utils"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	ops "github.com/findy-network/findy-common-go/grpc/ops/v1"
	"github.com/findy-network/findy-common-go/jwt"
	"github.com/findy-network/findy-common-go/rpc"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"google.golang.org/grpc"
)

var Server *grpc.Server

func Serve(conf *rpc.ServerCfg) {
	assert.D.True(conf != nil, "GRPC server needs configuration")
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

func taskFrom(protocol *pb.Protocol) (t comm.Task, err error) {
	defer err2.Return(&err)

	header := &comm.TaskHeader{
		TaskID:         utils.UUID(),
		TypeID:         uniqueTypeID(protocol.Role, protocol.TypeID),
		ProtocolTypeID: protocol.GetTypeID().String(),
		ProtocolRole:   protocol.GetRole().String(),
		ConnID:         protocol.GetConnectionID(),
	}
	return prot.CreateTask(header, protocol)
}

var notificationTypeID = map[string]pb.Notification_Type{
	pltype.CANotifyStatus:     pb.Notification_STATUS_UPDATE,
	pltype.CANotifyUserAction: pb.Notification_PROTOCOL_PAUSED,
}

var questionTypeID = map[string]pb.Question_Type{
	pltype.SAPing:                         pb.Question_PING_WAITS,
	pltype.SAIssueCredentialAcceptPropose: pb.Question_ISSUE_PROPOSE_WAITS,
	pltype.SAPresentProofAcceptPropose:    pb.Question_PROOF_PROPOSE_WAITS,
	pltype.SAPresentProofAcceptValues:     pb.Question_PROOF_VERIFY_WAITS,
}

func uniqueTypeID(role pb.Protocol_Role, id pb.Protocol_Type) string {
	i := int32(10*role) + int32(id)
	glog.V(5).Infoln("unique id:", i, typeID[i])
	s, ok := typeID[i]
	assert.D.Truef(ok, "cannot find typeid for (%d+%d)", role, id)
	return s
}

// TODO: Should we shift for `role` and consider what happens when w3c protocols
// come along

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

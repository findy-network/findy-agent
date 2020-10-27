/*
Package server is a playground where we have tried gRPC interface for CA API.
*/
package server

import (
	"context"
	"errors"
	"os"
	"path"

	pb "github.com/findy-network/findy-agent-api/grpc/agency"
	"github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
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
	goPath := os.Getenv("GOPATH")
	tlsPath := path.Join(goPath, "src/github.com/findy-network/findy-grpc/tls")
	certFile := path.Join(tlsPath, "server.crt")
	keyFile := path.Join(tlsPath, "server.pem")

	glog.V(1).Infoln("starting gRPC server with tls path:", tlsPath)

	rpc.Serve(rpc.ServerCfg{
		Port:     50051,
		TLS:      true,
		CertFile: certFile,
		KeyFile:  keyFile,
		Register: func(s *grpc.Server) error {
			pb.RegisterDIDCommServer(s, &didCommServer{})
			pb.RegisterAgentServer(s, &agentServer{})
			return nil
		},
	})
}

type agentServer struct {
	pb.UnimplementedAgentServer
}

func (a *agentServer) Give(ctx context.Context, answer *pb.Answer) (cid *pb.ClientID, err error) {
	defer err2.Annotate("give answer", &err)

	glog.V(0).Info("Give function start")

	caDID, receiver := e2.StrRcvr.Try(ca(ctx))

	glog.V(0).Infoln(caDID, "-agent answers:", answer.ClientId.Id, answer.Id)
	bus.WantAllAgentAnswers.AgentSendAnswer(bus.AgentAnswer{
		ID: answer.Id,
		AgentKeyType: bus.AgentKeyType{
			AgentDID: receiver.WDID(),
			ClientID: answer.ClientId.Id,
		},
		ACK:  true,
		Info: "welcome from gRPC",
	})

	return &pb.ClientID{Id: ""}, nil
}

func (a *agentServer) Listen(clientID *pb.ClientID, server pb.Agent_ListenServer) (err error) {
	defer err2.Handle(&err, func() {
		glog.Errorf("grpc agent listen error: %s", err)
		status := &pb.AgentStatus{
			ClientId: &pb.ClientID{Id: clientID.Id},
		}
		if err := server.Send(status); err != nil {
			glog.Errorln("error sending response:", err)
		}
	})

	ctx := e2.Ctx.Try(jwt.CheckTokenValidity(server.Context()))
	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	glog.V(0).Infoln(caDID, "-agent starts listener:", clientID.Id)

	listenKey := bus.AgentKeyType{
		AgentDID: receiver.WDID(),
		ClientID: clientID.Id,
	}
	notifyChan := bus.WantAllAgentActions.AgentAddListener(listenKey)
	defer bus.WantAllAgentActions.AgentRmListener(listenKey)
	questionChan := bus.WantAllAgentAnswers.AgentAddAnswerer(listenKey)
	defer bus.WantAllAgentAnswers.AgentRmAnswerer(listenKey)

loop:
	for {
		//var notify bus.AgentNotify
		select {
		case question := <-questionChan:
			glog.V(3).Infoln("question arrived", question.ID)
			agentStatus := pb.AgentStatus{
				ClientId: &pb.ClientID{Id: question.AgentDID},
				Notification: &pb.Notification{
					Id:             question.ID,
					TypeId:         notificationTypeID[question.NotificationType],
					ConnectionId:   question.ConnectionID,
					ProtocolId:     question.ProtocolID,
					ProtocolFamily: question.ProtocolFamily,
					Timestamp:      question.TimestampMs,
				},
			}
			if clientID.Id != question.ClientID {
				glog.Warningf("client id mismatch: c/s: %s/%s",
					clientID.Id, question.ClientID)
			}
			agentStatus.ClientId.Id = question.ClientID
			err2.Check(server.Send(&agentStatus))

		case notify := <-notifyChan:
			glog.V(3).Infoln("notification", notify.ID, "arrived")
			agentStatus := pb.AgentStatus{
				ClientId: &pb.ClientID{Id: notify.AgentDID},
				Notification: &pb.Notification{
					Id:             notify.ID,
					TypeId:         notificationTypeID[notify.NotificationType],
					ConnectionId:   notify.ConnectionID,
					ProtocolId:     notify.ProtocolID,
					ProtocolFamily: notify.ProtocolFamily,
					Timestamp:      notify.TimestampMs,
				},
			}
			if clientID.Id != notify.ClientID {
				glog.Warningf("client id mismatch: c/s: %s/%s",
					clientID.Id, notify.ClientID)
			}
			agentStatus.ClientId.Id = notify.ClientID
			err2.Check(server.Send(&agentStatus))
		case <-ctx.Done():
			glog.V(1).Infoln("ctx.Done() received, returning")
			break loop
		}
	}
	return nil
}

type didCommServer struct {
	pb.UnimplementedDIDCommServer
}

func (s *didCommServer) Start(ctx context.Context, protocol *pb.Protocol) (pid *pb.ProtocolID, err error) {
	defer err2.Return(&err)

	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	task := e2.Task.Try(taskFrom(protocol))
	glog.V(0).Infoln(caDID, "-agent starts protocol:", pb.Protocol_Type_name[int32(protocol.TypeId)])
	prot.FindAndStartTask(receiver, task)
	return &pb.ProtocolID{Id: task.Nonce}, nil
}

func (s *didCommServer) Status(ctx context.Context, id *pb.ProtocolID) (ps *pb.ProtocolStatus, err error) {
	defer err2.Return(&err)

	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	task := &comm.Task{
		Nonce:  id.Id,
		TypeID: typeID[id.TypeId],
	}
	key := psm.NewStateKey(receiver.WorkerEA(), task.Nonce)
	glog.V(0).Infoln(caDID, "-agent protocol status:", pb.Protocol_Type_name[int32(id.TypeId)])
	statusJSON := dto.ToJSON(prot.GetStatus(task.TypeID, &key))

	return &pb.ProtocolStatus{
		State:   &pb.ProtocolState{ProtocolId: id},
		Message: statusJSON,
	}, nil
}

func (s *didCommServer) Run(protocol *pb.Protocol, server pb.DIDComm_RunServer) (err error) {
	defer err2.Handle(&err, func() {
		glog.Errorf("grpc run error: %s", err)
		status := &pb.ProtocolState{
			Info:  err.Error(),
			State: pb.ProtocolState_ERR,
		}
		if err := server.Send(status); err != nil {
			glog.Errorln("error sending response:", err)
		}
	})

	ctx := e2.Ctx.Try(jwt.CheckTokenValidity(server.Context()))
	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	task := e2.Task.Try(taskFrom(protocol))
	glog.V(0).Infoln(caDID, "-agent starts protocol:", pb.Protocol_Type_name[int32(protocol.TypeId)])

	key := psm.NewStateKey(receiver.WorkerEA(), task.Nonce)
	statusChan := bus.WantAll.AddListener(key)
	userActionChan := bus.WantUserActions.AddListener(key)

	prot.FindAndStartTask(receiver, task)

	var statusCode pb.ProtocolState_State
loop:
	for {
		select {
		case status := <-statusChan:
			glog.V(1).Infoln("grpc state:", status)
			switch status {
			case psm.ReadyACK, psm.ACK:
				statusCode = pb.ProtocolState_OK
				break loop
			case psm.ReadyNACK, psm.NACK, psm.Failure:
				statusCode = pb.ProtocolState_ERR
				break loop
			}
		case status := <-userActionChan:
			switch status {
			case psm.Waiting:
				glog.V(1).Infoln("waiting arrived")
				status := &pb.ProtocolState{
					ProtocolId: &pb.ProtocolID{Id: task.Nonce},
					State:      pb.ProtocolState_WAIT_ACTION,
				}
				err2.Check(server.Send(status))
			}
		}
	}
	glog.V(1).Infoln("out from grpc state:", statusCode)
	bus.WantAll.RmListener(key)
	bus.WantUserActions.RmListener(key)

	status := &pb.ProtocolState{
		ProtocolId: &pb.ProtocolID{Id: task.Nonce},
		State:      statusCode,
	}
	err2.Check(server.Send(status))

	return nil
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
	pltype.CAPairwiseCreate, // "CONNECT",
	pltype.CACredOffer,      // "ISSUE",
	pltype.CACredRequest,    // "PROPOSE_ISSUING",
	pltype.CAProofRequest,   // "REQUEST_PROOF",
	pltype.CAProofPropose,   // "PROPOSE_PROOFING",
	pltype.CATrustPing,      // "TRUST_PING",
	pltype.CABasicMessage,   // "BASIC_MESSAGE",
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

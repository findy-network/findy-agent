package server

import (
	"context"

	pb "github.com/findy-network/findy-agent-api/grpc/agency"
	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/utils"
	didexchange "github.com/findy-network/findy-agent/std/didexchange/invitation"
	"github.com/findy-network/findy-grpc/jwt"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

type agentServer struct {
	pb.UnimplementedAgentServer
}

func (a *agentServer) SetImplId(ctx context.Context, implementation *pb.SAImplementation) (impl *pb.SAImplementation, err error) {
	defer err2.Annotate("set impl", &err)

	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	glog.V(1).Infoln(caDID, "-agent set impl:", implementation.Id)
	receiver.AttachSAImpl(implementation.Id)
	return &pb.SAImplementation{Id: implementation.Id}, nil
}

func (a *agentServer) CreateInvitation(ctx context.Context, base *pb.InvitationBase) (inv *pb.Invitation, err error) {
	defer err2.Annotate("create invitation", &err)

	_, receiver := e2.StrRcvr.Try(ca(ctx))
	ep := receiver.CAEndp(true)
	ep.RcvrDID = receiver.Trans().MessagePipe().Out.Did()

	id := base.Id
	if id == "" {
		id = utils.UUID()
	}
	label := base.Label
	if base.Label == "" {
		label = "empty-label"
	}
	invitation := didexchange.Invitation{
		ID:              id,
		Type:            pltype.AriesConnectionInvitation,
		ServiceEndpoint: ep.Address(),
		RecipientKeys:   []string{receiver.Trans().PayloadPipe().In.VerKey()},
		Label:           label,
	}

	jStr := dto.ToJSON(invitation)

	return &pb.Invitation{JsonStr: jStr}, nil
}

func (a *agentServer) Give(ctx context.Context, answer *pb.Answer) (cid *pb.ClientID, err error) {
	defer err2.Annotate("give answer", &err)

	_, receiver := e2.StrRcvr.Try(ca(ctx))
	bus.WantAllAgentAnswers.AgentSendAnswer(bus.AgentAnswer{
		ID: answer.Id,
		AgentKeyType: bus.AgentKeyType{
			AgentDID: receiver.WDID(),
			ClientID: answer.ClientId.Id,
		},
		ACK:  answer.Ack,
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
	glog.V(1).Infoln(caDID, "-agent starts listener:", clientID.Id)

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
			glog.V(1).Infoln("QUESTION ARRIVED", question.ID)

			agentStatus := pb.AgentStatus{
				ClientId: &pb.ClientID{Id: question.AgentDID},
				Notification: &pb.Notification{
					Id:             question.ID,
					TypeId:         notificationTypeID[question.NotificationType],
					ConnectionId:   question.ConnectionID,
					ProtocolId:     question.ProtocolID,
					ProtocolFamily: question.ProtocolFamily,
					ProtocolType:   protocolType[question.ProtocolFamily],
					Timestamp:      question.Timestamp,
					Role:           roleType[question.Initiator],
				},
			}
			if clientID.Id != question.ClientID {
				glog.Warningf("client id mismatch: c/s: %s/%s",
					clientID.Id, question.ClientID)
			}
			agentStatus.ClientId.Id = question.ClientID
			err2.Check(server.Send(&agentStatus))

		case notify := <-notifyChan:
			glog.V(1).Infoln("notification", notify.ID, "arrived")
			agentStatus := pb.AgentStatus{
				ClientId: &pb.ClientID{Id: notify.AgentDID},
				Notification: &pb.Notification{
					Id:             notify.ID,
					TypeId:         notificationTypeID[notify.NotificationType],
					ConnectionId:   notify.ConnectionID,
					ProtocolId:     notify.ProtocolID,
					ProtocolFamily: notify.ProtocolFamily,
					ProtocolType:   protocolType[notify.ProtocolFamily],
					Timestamp:      notify.Timestamp,
					Role:           roleType[notify.Initiator],
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

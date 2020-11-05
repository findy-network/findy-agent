package server

import (
	"context"

	pb "github.com/findy-network/findy-agent-api/grpc/agency"
	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-grpc/jwt"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

type agentServer struct {
	pb.UnimplementedAgentServer
}

func (a *agentServer) Give(ctx context.Context, answer *pb.Answer) (cid *pb.ClientID, err error) {
	defer err2.Annotate("give answer", &err)

	glog.V(1).Info("Give function start")

	caDID, receiver := e2.StrRcvr.Try(ca(ctx))

	glog.V(1).Infoln(caDID, "-agent answers:", answer.ClientId.Id, answer.Id)
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

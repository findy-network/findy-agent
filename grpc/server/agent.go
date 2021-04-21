package server

import (
	"context"
	"time"

	pb "github.com/findy-network/findy-agent-api/grpc/agency"
	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	didexchange "github.com/findy-network/findy-agent/std/didexchange/invitation"
	"github.com/findy-network/findy-common-go/jwt"
	"github.com/findy-network/findy-wrapper-go"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/findy-network/findy-wrapper-go/ledger"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
)

const keepaliveTimer = 50 * time.Second

type agentServer struct {
	pb.UnimplementedAgentServer
}

func (a *agentServer) Ping(
	ctx context.Context,
	pm *pb.PingMsg,
) (
	_ *pb.PingMsg,
	err error,
) {
	defer err2.Annotate("agent server ping", &err)

	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	glog.V(1).Infoln(caDID, "-agent ping:", pm.Id)

	saReply := false
	if pm.PingController {
		glog.V(3).Info("calling sa ping")
		om := mesg.MsgCreator.Create(didcomm.MsgInit{}).(didcomm.Msg)
		ask, err := receiver.CallEA(pltype.SAPing, om)
		err2.Check(err)
		saReply = ask.Ready()
	}
	return &pb.PingMsg{Id: pm.Id, PingController: saReply}, nil
}

func (a *agentServer) CreateSchema(
	ctx context.Context,
	s *pb.SchemaCreate,
) (
	os *pb.Schema,
	err error,
) {
	defer err2.Annotate("create schema", &err)

	caDID, ca := e2.StrRcvr.Try(ca(ctx))
	glog.V(1).Infoln(caDID, "-agent create schema:", s.Name)

	sch := &ssi.Schema{
		Name:    s.Name,
		Version: s.Version,
		Attrs:   s.Attrs,
	}
	err2.Check(sch.Create(ca.RootDid().Did()))
	err2.Check(sch.ToLedger(ca.Wallet(), ca.RootDid().Did()))

	return &pb.Schema{Id: sch.ValidID()}, nil
}

func (a *agentServer) CreateCredDef(
	ctx context.Context,
	cdc *pb.CredDefCreate,
) (
	_ *pb.CredDef,
	err error,
) {
	defer err2.Annotate("create creddef", &err)

	caDID, ca := e2.StrRcvr.Try(ca(ctx))
	glog.V(1).Infoln(caDID, "-agent create creddef:", cdc.Tag,
		"schema:", cdc.SchemaId)

	sch := &ssi.Schema{ID: cdc.SchemaId}
	err2.Check(sch.FromLedger(ca.RootDid().Did()))
	r := <-anoncreds.IssuerCreateAndStoreCredentialDef(
		ca.Wallet(), ca.RootDid().Did(), sch.Stored.Str2(),
		cdc.Tag, findy.NullString, findy.NullString)
	err2.Check(r.Err())
	cd := r.Str2()
	err = ledger.WriteCredDef(ca.Pool(), ca.Wallet(), ca.RootDid().Did(), cd)
	return &pb.CredDef{Id: r.Str1()}, nil
}

func (a *agentServer) GetSchema(
	ctx context.Context,
	s *pb.Schema,
) (
	_ *pb.SchemaData,
	err error,
) {
	defer err2.Annotate("get schema", &err)

	caDID, ca := e2.StrRcvr.Try(ca(ctx))
	glog.V(1).Infoln(caDID, "-agent get schema:", s.Id)

	sID, schema, err := ledger.ReadSchema(ca.Pool(), ca.RootDid().Did(), s.Id)
	err2.Check(err)
	return &pb.SchemaData{Id: sID, Data: schema}, nil
}

func (a *agentServer) GetCredDef(
	ctx context.Context,
	cd *pb.CredDef,
) (
	_ *pb.CredDefData,
	err error,
) {
	defer err2.Annotate("get creddef", &err)

	caDID, ca := e2.StrRcvr.Try(ca(ctx))
	glog.V(1).Infoln(caDID, "-agent get creddef:", cd.Id)

	def := err2.String.Try(ssi.CredDefFromLedger(ca.RootDid().Did(), cd.Id))
	return &pb.CredDefData{Id: cd.Id, Data: def}, nil
}

func (a *agentServer) SetImplId(ctx context.Context, implementation *pb.SAImplementation) (impl *pb.SAImplementation, err error) {
	defer err2.Annotate("set impl", &err)

	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	glog.V(1).Infoln(caDID, "-agent set impl:", implementation.Id)
	receiver.AttachSAImpl(implementation.Id, implementation.Persistent)
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

	return &pb.ClientID{Id: answer.ClientId.Id}, nil
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
		select {
		case question := <-questionChan:
			glog.V(1).Infoln("QUESTION in conn-id:", question.ConnectionID,
				"QID:", question.ID,
				question.ProtocolFamily,
				notificationTypeID[question.NotificationType])
			assert.D.True(clientID.Id == question.ClientID)
			agentStatus := processQuestion(question)
			agentStatus.ClientId.Id = question.ClientID
			err2.Check(server.Send(agentStatus))
			glog.V(1).Infoln("send question..")

		case notify := <-notifyChan:
			glog.V(1).Infoln("notification", notify.ID, "arrived")
			assert.D.True(clientID.Id == notify.ClientID)
			agentStatus := processNofity(notify)
			agentStatus.ClientId.Id = notify.ClientID
			err2.Check(server.Send(agentStatus))

		case <-time.After(keepaliveTimer):
			// send keep alive message
			glog.V(7).Infoln("sending keepalive timer")
			err2.Check(server.Send(&pb.AgentStatus{
				ClientId: &pb.ClientID{Id: clientID.Id},
				Notification: &pb.Notification{
					TypeId: pb.Notification_KEEPALIVE,
				}}))

		case <-ctx.Done():
			glog.V(1).Infoln("ctx.Done() received, returning")
			break loop
		}
	}
	return nil
}

func processQuestion(question bus.AgentQuestion) (as *pb.AgentStatus) {
	agentStatus := pb.AgentStatus{
		ClientId: &pb.ClientID{Id: question.AgentDID},
		Notification: &pb.Notification{
			Id:             question.ID,
			PID:            question.PID,
			TypeId:         notificationTypeID[question.NotificationType],
			ConnectionId:   question.ConnectionID,
			ProtocolId:     question.ProtocolID,
			ProtocolFamily: question.ProtocolFamily,
			ProtocolType:   protocolType[question.ProtocolFamily],
			Timestamp:      question.Timestamp,
			Role:           roleType[question.Initiator],
		},
	}
	if question.IssuePropose != nil {
		glog.V(1).Infoln("issue propose handling")
		agentStatus.Notification.Question = &pb.Notification_IssuePropose_{
			IssuePropose: &pb.Notification_IssuePropose{
				CredDefId:  question.IssuePropose.CredDefID,
				ValuesJson: question.IssuePropose.ValuesJSON,
			},
		}
	}
	if question.ProofVerify != nil {
		glog.V(1).Infoln("proof verify handling")
		attrs := make([]*pb.Notification_ProofVerify_Attr, 0, len(question.ProofVerify.Attrs))
		for _, attr := range question.Attrs {
			attrs = append(attrs, &pb.Notification_ProofVerify_Attr{
				Value:     attr.Value,
				Name:      attr.Name,
				CredDefId: attr.CredDefID,
				Predicate: attr.Predicate,
			})
		}
		agentStatus.Notification.Question = &pb.Notification_ProofVerify_{
			ProofVerify: &pb.Notification_ProofVerify{
				Attrs: attrs,
			},
		}
	}
	return &agentStatus
}

func processNofity(notify bus.AgentNotify) (as *pb.AgentStatus) {
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
	return &agentStatus
}

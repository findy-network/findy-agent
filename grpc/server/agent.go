package server

import (
	"context"
	"time"

	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	didexchange "github.com/findy-network/findy-agent/std/didexchange/invitation"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
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
	pb.UnimplementedAgentServiceServer
}

func (a *agentServer) Enter(
	ctx context.Context,
	mode *pb.ModeCmd,
) (
	rm *pb.ModeCmd,
	err error,
) {
	defer err2.Annotate("agent server enter mode cmd", &err)

	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	glog.V(1).Infoln(caDID, "-agent enter mode:", mode.TypeID, mode.IsInput)

	retMode := pb.ModeCmd_AcceptModeCmd_DEFAULT

	setModeFn := func(m pb.ModeCmd_AcceptModeCmd_Mode) *pb.ModeCmd {
		glog.V(3).Infoln("setModeFn:", m)
		return &pb.ModeCmd{
			TypeID: pb.ModeCmd_ACCEPT_MODE,
			ControlCmd: &pb.ModeCmd_AcceptMode{
				AcceptMode: &pb.ModeCmd_AcceptModeCmd{
					Mode: m,
				},
			},
		}
	}
	rm = setModeFn(retMode)

	if mode.IsInput {
		switch mode.TypeID {
		case pb.ModeCmd_ACCEPT_MODE:
			switch mode.GetAcceptMode().GetMode() {
			case pb.ModeCmd_AcceptModeCmd_AUTO_ACCEPT:
				glog.V(3).Infoln("--- Setting auto accept mode")
				receiver.AttachSAImpl("permissive_sa", false)
				rm = setModeFn(pb.ModeCmd_AcceptModeCmd_AUTO_ACCEPT)
			case pb.ModeCmd_AcceptModeCmd_GRPC_CONTROL:
				glog.V(3).Infoln("--- Setting default mode")
				receiver.AttachSAImpl("grpc", false)
				rm = setModeFn(pb.ModeCmd_AcceptModeCmd_GRPC_CONTROL)
			default:
				glog.V(3).Infoln("--- Setting default mode")
				receiver.AttachSAImpl("grpc", false)
			}
		}
	} else {
		if mode.TypeID == pb.ModeCmd_ACCEPT_MODE {
			if receiver.AutoPermission() {
				rm = setModeFn(pb.ModeCmd_AcceptModeCmd_AUTO_ACCEPT)
			}
		}
	}

	return rm, nil
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
	glog.V(1).Infoln(caDID, "-agent ping:", pm.ID)

	saReply := false
	if pm.PingController {
		glog.V(3).Info("calling sa ping")
		om := mesg.MsgCreator.Create(didcomm.MsgInit{}).(didcomm.Msg)
		ask, err := receiver.CallEA(pltype.SAPing, om)
		err2.Check(err)
		saReply = ask.Ready()
	}
	return &pb.PingMsg{ID: pm.ID, PingController: saReply}, nil
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
		Attrs:   s.Attributes,
	}
	err2.Check(sch.Create(ca.RootDid().Did()))
	err2.Check(sch.ToLedger(ca.Wallet(), ca.RootDid().Did()))

	return &pb.Schema{ID: sch.ValidID()}, nil
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
		"schema:", cdc.SchemaID)

	sch := &ssi.Schema{ID: cdc.SchemaID}
	err2.Check(sch.FromLedger(ca.RootDid().Did()))
	r := <-anoncreds.IssuerCreateAndStoreCredentialDef(
		ca.Wallet(), ca.RootDid().Did(), sch.Stored.Str2(),
		cdc.Tag, findy.NullString, findy.NullString)
	err2.Check(r.Err())
	cd := r.Str2()
	err = ledger.WriteCredDef(ca.Pool(), ca.Wallet(), ca.RootDid().Did(), cd)
	return &pb.CredDef{ID: r.Str1()}, nil
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
	glog.V(1).Infoln(caDID, "-agent get schema:", s.ID)

	sID, schema, err := ledger.ReadSchema(ca.Pool(), ca.RootDid().Did(), s.ID)
	err2.Check(err)
	return &pb.SchemaData{ID: sID, Data: schema}, nil
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
	glog.V(1).Infoln(caDID, "-agent get creddef:", cd.ID)

	def := err2.String.Try(ssi.CredDefFromLedger(ca.RootDid().Did(), cd.ID))
	return &pb.CredDefData{ID: cd.ID, Data: def}, nil
}

func (a *agentServer) CreateInvitation(ctx context.Context, base *pb.InvitationBase) (inv *pb.Invitation, err error) {
	defer err2.Annotate("create invitation", &err)

	id := base.ID
	if id == "" {
		id = utils.UUID()
	}
	_, receiver := e2.StrRcvr.Try(ca(ctx))
	ep := receiver.CAEndp(true)
	ep.RcvrDID = receiver.Trans().MessagePipe().Out.Did()
	ep.EdgeToken = id
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

	return &pb.Invitation{JSON: jStr}, nil
}

func (a *agentServer) Give(ctx context.Context, answer *pb.Answer) (cid *pb.ClientID, err error) {
	defer err2.Annotate("give answer", &err)

	_, receiver := e2.StrRcvr.Try(ca(ctx))
	bus.WantAllAgentAnswers.AgentSendAnswer(bus.AgentAnswer{
		ID: answer.ID,
		AgentKeyType: bus.AgentKeyType{
			AgentDID: receiver.WDID(),
			ClientID: answer.ClientID.ID,
		},
		ACK:  answer.Ack,
		Info: "welcome from gRPC",
	})

	return &pb.ClientID{ID: answer.ClientID.ID}, nil
}

func (a *agentServer) Listen(clientID *pb.ClientID, server pb.AgentService_ListenServer) (err error) {
	defer err2.Handle(&err, func() {
		glog.Errorf("grpc agent listen error: %s", err)
		status := &pb.AgentStatus{
			ClientID: &pb.ClientID{ID: clientID.ID},
		}
		if err := server.Send(status); err != nil {
			glog.Errorln("error sending response:", err)
		}
	})

	ctx := e2.Ctx.Try(jwt.CheckTokenValidity(server.Context()))
	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	glog.V(1).Infoln(caDID, "-agent starts listener:", clientID.ID)

	listenKey := bus.AgentKeyType{
		AgentDID: receiver.WDID(),
		ClientID: clientID.ID,
	}
	notifyChan := bus.WantAllAgentActions.AgentAddListener(listenKey)
	defer bus.WantAllAgentActions.AgentRmListener(listenKey)

loop:
	for {
		select {
		case notify := <-notifyChan:
			glog.V(1).Infoln("notification", notify.ID, "arrived")
			assert.D.True(clientID.ID == notify.ClientID)
			agentStatus := processNofity(notify)
			agentStatus.ClientID.ID = notify.ClientID
			err2.Check(server.Send(agentStatus))

		case <-time.After(keepaliveTimer):
			// send keep alive message
			glog.V(7).Infoln("sending keepalive timer")
			err2.Check(server.Send(&pb.AgentStatus{
				ClientID: &pb.ClientID{ID: clientID.ID},
				Notification: &pb.Notification{
					TypeID: pb.Notification_KEEPALIVE,
				}}))

		case <-ctx.Done():
			glog.V(1).Infoln("ctx.Done() received, returning")
			break loop
		}
	}
	return nil
}

func (a *agentServer) Wait(clientID *pb.ClientID, server pb.AgentService_WaitServer) (err error) {
	defer err2.Handle(&err, func() {
		glog.Errorf("grpc agent listen error: %s", err)
		status := &pb.Question{
			Status: &pb.AgentStatus{
				ClientID: &pb.ClientID{ID: clientID.ID},
			},
		}
		if err := server.Send(status); err != nil {
			glog.Errorln("error sending response:", err)
		}
	})

	ctx := e2.Ctx.Try(jwt.CheckTokenValidity(server.Context()))
	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	glog.V(1).Infoln(caDID, "-agent starts Wait():", clientID.ID)

	listenKey := bus.AgentKeyType{
		AgentDID: receiver.WDID(),
		ClientID: clientID.ID,
	}
	questionChan := bus.WantAllAgentAnswers.AgentAddAnswerer(listenKey)
	defer bus.WantAllAgentAnswers.AgentRmAnswerer(listenKey)

loop:
	for {
		select {
		case question := <-questionChan:
			glog.V(1).Infoln("QUESTION in conn-id:", question.ConnectionID,
				"QID:", question.ID,
				question.ProtocolFamily,
				questionTypeID[question.NotificationType])
			assert.D.True(clientID.ID == question.ClientID)
			q2send := processQuestion(question)
			q2send.Status.ClientID.ID = question.ClientID
			err2.Check(server.Send(q2send))
			glog.V(1).Infoln("send question..")

		case <-time.After(keepaliveTimer):
			// send keep alive message
			glog.V(7).Infoln("sending keepalive timer")
			err2.Check(server.Send(&pb.Question{
				Status: &pb.AgentStatus{
					ClientID: &pb.ClientID{ID: clientID.ID},
				},
				TypeID: pb.Question_KEEPALIVE,
			}))

		case <-ctx.Done():
			glog.V(1).Infoln("ctx.Done() received, returning")
			break loop
		}
	}
	return nil
}

func processQuestion(question bus.AgentQuestion) (as *pb.Question) {
	q2send := pb.Question{
		TypeID: questionTypeID[question.NotificationType],
		Status: &pb.AgentStatus{
			ClientID: &pb.ClientID{ID: question.AgentDID},
			Notification: &pb.Notification{
				ID:             question.ID,
				PID:            question.PID,
				ConnectionID:   question.ConnectionID,
				ProtocolID:     question.ProtocolID,
				ProtocolFamily: question.ProtocolFamily,
				ProtocolType:   protocolType[question.ProtocolFamily],
				Timestamp:      question.Timestamp,
				Role:           roleType[question.Initiator],
			},
		},
	}
	if question.IssuePropose != nil {
		glog.V(1).Infoln("issue propose handling")
		q2send.Question = &pb.Question_IssuePropose{
			IssuePropose: &pb.Question_IssueProposeMsg{
				CredDefID:  question.IssuePropose.CredDefID,
				ValuesJSON: question.IssuePropose.ValuesJSON,
			},
		}
	}
	if question.ProofVerify != nil {
		glog.V(1).Infoln("proof verify handling")
		attrs := make([]*pb.Question_ProofVerifyMsg_Attribute, 0,
			len(question.ProofVerify.Attrs))
		for _, attr := range question.Attrs {
			attrs = append(attrs, &pb.Question_ProofVerifyMsg_Attribute{
				Value:     attr.Value,
				Name:      attr.Name,
				CredDefID: attr.CredDefID,
				//Predicate: attr.Predicate,
			})
		}
		q2send.Question = &pb.Question_ProofVerify{
			ProofVerify: &pb.Question_ProofVerifyMsg{
				Attributes: attrs,
			},
		}
	}
	return &q2send
}

func processNofity(notify bus.AgentNotify) (as *pb.AgentStatus) {
	agentStatus := pb.AgentStatus{
		ClientID: &pb.ClientID{ID: notify.AgentDID},
		Notification: &pb.Notification{
			ID:             notify.ID,
			TypeID:         notificationTypeID[notify.NotificationType],
			ConnectionID:   notify.ConnectionID,
			ProtocolID:     notify.ProtocolID,
			ProtocolFamily: notify.ProtocolFamily,
			ProtocolType:   protocolType[notify.ProtocolFamily],
			Timestamp:      notify.Timestamp,
			Role:           roleType[notify.Initiator],
		},
	}
	return &agentStatus
}

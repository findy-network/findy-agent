package server

import (
	"context"
	"time"

	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
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
		glog.V(3).Info("calling sa ping", receiver.WDID())
		if receiver.AutoPermission() {
			saReply = true
		} else {
			// TODO:
			// we cannot return ping answer currently synchronously
			// if needed:
			// 1) create task with user action handling for ping
			// 2) block while answer is received
			// 3) add ping support to Resume-API
			glog.Warning("SA ping not implemented")
		}
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

	// in the future we might have generatedConnID as argument in the API
	id := base.ID
	generatedConnID := utils.UUID()
	// if connection is not given from the caller we generate a new one and use
	// it for both one
	if id == "" {
		id = utils.UUID()
		generatedConnID = id
		glog.V(4).Infoln("generating connection id:", id)
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
		ID:              generatedConnID,
		Type:            pltype.AriesConnectionInvitation,
		ServiceEndpoint: ep.Address(),
		RecipientKeys:   []string{receiver.Trans().PayloadPipe().In.VerKey()},
		Label:           label,
	}

	jStr := dto.ToJSON(invitation)

	// TODO: add connection id to return struct as well, gRPC API Change
	return &pb.Invitation{JSON: jStr}, nil
}

func (a *agentServer) Give(ctx context.Context, answer *pb.Answer) (cid *pb.ClientID, err error) {
	defer err2.Annotate("give answer", &err)

	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	glog.V(1).Infoln(caDID, "-agent Give/Resume protocol:",
		answer.ID,
		answer.Ack,
	)

	state, err := psm.GetPSM(psm.StateKey{
		DID:   caDID,
		Nonce: answer.ID,
	})
	err2.Check(err)

	prot.Resume(
		receiver,
		uniqueTypeID(pb.Protocol_RESUMER, state.FirstState().T.ProtocolType()),
		answer.ID, answer.Ack)

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
		if err = server.Send(status); err != nil {
			glog.Errorln("error sending response:", err)
		}
	})

	ctx := e2.Ctx.Try(jwt.CheckTokenValidity(server.Context()))
	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	glog.V(1).Infoln(caDID, "-agent starts listener:", clientID.ID)

	waitClientID := "Wait" + clientID.ID // avoid collisions
	listenKey := bus.AgentKeyType{
		AgentDID: receiver.WDID(),
		ClientID: waitClientID,
	}

	notifyChan := bus.WantAllAgentActions.AgentAddListener(listenKey)
	defer bus.WantAllAgentActions.AgentRmListener(listenKey)

loop:
	for {
		select {
		case notify := <-notifyChan:
			glog.V(1).Infoln("notification", notify.ID, "arrived")
			assert.D.True(waitClientID == notify.ClientID)

			var question *pb.Question
			question, err = processQuestion(ctx, notify)
			err2.Check(err)
			if question != nil {
				question.Status.ClientID.ID = clientID.ID
				err2.Check(server.Send(question))
				glog.V(1).Infoln("send question..")
			}

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

func processQuestion(ctx context.Context, notify bus.AgentNotify) (as *pb.Question, err error) {
	defer err2.Annotate("processQuestion", &err)

	notificationType := notificationForPayloadType(notify.NotificationType)
	notificationProtocolType := pltype.ProtocolTypeForFamily(notify.ProtocolFamily)

	if notificationType != pb.Notification_PROTOCOL_PAUSED {
		return nil, nil
	}

	glog.V(3).Infof("Sending question with id %s", notify.ProtocolID)

	q2send := pb.Question{
		TypeID: questionTypeID[notify.NotificationType],
		Status: &pb.AgentStatus{
			ClientID: &pb.ClientID{ID: notify.AgentDID},
			Notification: &pb.Notification{
				ID:             notify.ProtocolID,
				PID:            notify.ProtocolID,
				ConnectionID:   notify.ConnectionID,
				ProtocolID:     notify.ProtocolID,
				ProtocolFamily: notify.ProtocolFamily,
				ProtocolType:   notificationProtocolType,
				Timestamp:      notify.Timestamp,
				Role:           roleType[notify.Initiator],
			},
		},
	}

	id := &pb.ProtocolID{
		TypeID: pltype.ProtocolTypeForFamily(notify.ProtocolFamily),
		Role:   roleType[notify.Initiator],
		ID:     notify.ProtocolID,
	}

	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	key := psm.NewStateKey(receiver.WorkerEA(), id.ID)
	glog.V(1).Infoln(caDID, "-agent protocol status:", pb.Protocol_Type_name[int32(id.TypeID)], protocolName[id.TypeID])
	ps, _ := tryProtocolStatus(id, key)

	switch notificationProtocolType {
	case pb.Protocol_ISSUE_CREDENTIAL:
		glog.V(1).Infoln("issue propose handling")
		q2send.Question = &pb.Question_IssuePropose{
			IssuePropose: &pb.Question_IssueProposeMsg{
				CredDefID:  ps.GetIssueCredential().GetCredDefID(),
				ValuesJSON: ps.GetIssueCredential().GetAttributes().String(), // TODO?
			},
		}
	case pb.Protocol_PRESENT_PROOF:
		glog.V(1).Infoln("proof verify handling")
		attrs := make([]*pb.Question_ProofVerifyMsg_Attribute, 0,
			len(ps.GetPresentProof().GetProof().Attributes))
		for _, attr := range ps.GetPresentProof().GetProof().Attributes {
			attrs = append(attrs, &pb.Question_ProofVerifyMsg_Attribute{
				Value:     attr.Value,
				Name:      attr.Name,
				CredDefID: attr.CredDefID,
			})
		}
		q2send.Question = &pb.Question_ProofVerify{
			ProofVerify: &pb.Question_ProofVerifyMsg{
				Attributes: attrs,
			},
		}

	}

	return &q2send, nil
}

func processNofity(notify bus.AgentNotify) (as *pb.AgentStatus) {
	agentStatus := pb.AgentStatus{
		ClientID: &pb.ClientID{ID: notify.AgentDID},
		Notification: &pb.Notification{
			ID:             notify.ID,
			TypeID:         notificationForPayloadType(notify.NotificationType),
			ConnectionID:   notify.ConnectionID,
			ProtocolID:     notify.ProtocolID,
			ProtocolFamily: notify.ProtocolFamily,
			ProtocolType:   pltype.ProtocolTypeForFamily(notify.ProtocolFamily),
			Timestamp:      notify.Timestamp,
			Role:           roleType[notify.Initiator],
		},
	}
	return &agentStatus
}

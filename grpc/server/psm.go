package server

import (
	"context"

	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/findy-network/findy-common-go/jwt"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

type didCommServer struct {
	pb.UnimplementedProtocolServiceServer
}

func (s *didCommServer) Run(protocol *pb.Protocol, server pb.ProtocolService_RunServer) (err error) {
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

	glog.V(3).Infoln("run() call")

	ctx := e2.Ctx.Try(jwt.CheckTokenValidity(server.Context()))
	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	task := e2.Task.Try(taskFrom(protocol))
	glog.V(3).Infoln(caDID, "-agent starts protocol:", pb.Protocol_Type_name[int32(protocol.TypeID)])

	key := psm.NewStateKey(receiver.WorkerEA(), task.Nonce)
	statusChan := bus.WantAll.AddListener(key)
	userActionChan := bus.WantUserActions.AddListener(key)

	prot.FindAndStartTask(receiver, task)

	var statusCode pb.ProtocolState_State
loop:
	for {
		select {
		case status := <-statusChan:
			glog.V(1).Infof("grpc %s state in %s",
				status, task.Nonce)
			switch status {
			case psm.ReadyACK, psm.ACK:
				statusCode = pb.ProtocolState_OK
				break loop
			case psm.ReadyNACK, psm.NACK:
				statusCode = pb.ProtocolState_NACK
				break loop
			case psm.Failure:
				statusCode = pb.ProtocolState_ERR
				break loop
			}
		case status := <-userActionChan:
			switch status {
			case psm.Waiting:
				glog.V(1).Infoln("waiting arrived")
				status := &pb.ProtocolState{
					ProtocolID: &pb.ProtocolID{ID: task.Nonce},
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
		ProtocolID: &pb.ProtocolID{ID: task.Nonce},
		State:      statusCode,
	}
	err2.Check(server.Send(status))

	return nil
}

func (s *didCommServer) Resume(ctx context.Context, state *pb.ProtocolState) (pid *pb.ProtocolID, err error) {
	defer err2.Return(&err)

	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	glog.V(1).Infoln(caDID, "-agent Resume protocol:", state.ProtocolID.TypeID, state.ProtocolID.ID)

	prot.Resume(receiver, uniqueTypeID(state.ProtocolID.Role, state.ProtocolID.TypeID),
		state.ProtocolID.ID, state.GetState() == pb.ProtocolState_ACK)

	return state.ProtocolID, nil
}

func (s *didCommServer) Release(ctx context.Context, id *pb.ProtocolID) (ps *pb.ProtocolID, err error) {
	defer err2.Return(&err)

	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	glog.V(1).Infoln(caDID, "-agent release protocol:", id.ID)
	key := psm.NewStateKey(receiver.WorkerEA(), id.ID)
	err2.Check(prot.AddAndSetFlagUpdatePSM(key, psm.Archiving, 0))
	glog.V(1).Infoln(caDID, "-agent release OK", id.ID)

	return id, nil
}

func (s *didCommServer) Start(ctx context.Context, protocol *pb.Protocol) (pid *pb.ProtocolID, err error) {
	defer err2.Return(&err)

	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	task := e2.Task.Try(taskFrom(protocol))
	glog.V(1).Infoln(caDID, "-agent starts protocol:", pb.Protocol_Type_name[int32(protocol.TypeID)])
	prot.FindAndStartTask(receiver, task)
	return &pb.ProtocolID{ID: task.Nonce}, nil
}

func (s *didCommServer) Status(ctx context.Context, id *pb.ProtocolID) (ps *pb.ProtocolStatus, err error) {
	defer err2.Return(&err)

	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	key := psm.NewStateKey(receiver.WorkerEA(), id.ID)
	glog.V(1).Infoln(caDID, "-agent protocol status:", pb.Protocol_Type_name[int32(id.TypeID)], protocolName[id.TypeID])
	ps, _ = tryProtocolStatus(id, key)
	return ps, nil
}

func tryProtocolStatus(id *pb.ProtocolID, key psm.StateKey) (ps *pb.ProtocolStatus, connID string) {
	statusJSON := dto.ToJSON(prot.GetStatus(protocolName[id.TypeID], &key))
	m := e2.PSM.Try(psm.GetPSM(key))
	state := &pb.ProtocolState{
		ProtocolID: id,
		State:      calcProtocolState(m),
	}
	connID = m.PairwiseName()
	if m != nil {
		state.ProtocolID.Role = roleType[m.Initiator]
	} else {
		glog.Warningf("cannot get protocol role for %s", key)
		state.ProtocolID.Role = pb.Protocol_UNKNOWN
	}
	ps = &pb.ProtocolStatus{
		State:      state,
		StatusJson: statusJSON,
	}
	switch id.TypeID {
	case pb.Protocol_DIDEXCHANGE:
		ps.Status = tryGetConnectStatus(id, key)
	case pb.Protocol_ISSUE_CREDENTIAL:
		ps.Status = tryGetIssueStatus(id, key)
	case pb.Protocol_PRESENT_PROOF:
		ps.Status = tryGetProofStatus(id, key)
	case pb.Protocol_TRUST_PING:
		ps.Status = tryGetTrustPingStatus(id, key)
	case pb.Protocol_BASIC_MESSAGE:
		ps.Status = tryGetBasicMessageStatus(id, key)

	}
	return ps, connID
}

func calcProtocolState(m *psm.PSM) pb.ProtocolState_State {
	if m != nil {
		if m.PendingUserAction() {
			return pb.ProtocolState_WAIT_ACTION
		}
		if last := m.LastState(); last != nil {
			switch last.Sub.Pure() {
			case psm.Ready, psm.Ready | psm.Archiving:
				if last.Sub&psm.ACK != 0 {
					return pb.ProtocolState_OK
				}
				return pb.ProtocolState_NACK
			case psm.Failure, psm.Failure | psm.Archiving:
				return pb.ProtocolState_ERR
			}
		}
	}
	return pb.ProtocolState_RUNNING
}

func tryGetConnectStatus(
	_ *pb.ProtocolID,
	key psm.StateKey) *pb.ProtocolStatus_DIDExchange {
	pw, err := psm.GetPairwiseRep(key)
	err2.Check(err)

	myDID := pw.Callee
	theirDID := pw.Caller
	theirEndpoint := pw.Caller.Endp

	if !myDID.My {
		myDID = pw.Caller
		theirDID = pw.Callee
		theirEndpoint = pw.Callee.Endp
	}

	return &pb.ProtocolStatus_DIDExchange{DIDExchange: &pb.ProtocolStatus_Connection{
		ID:            pw.Name,
		MyDid:         myDID.DID,
		TheirDid:      theirDID.DID,
		TheirEndpoint: theirEndpoint,
		TheirLabel:    pw.TheirLabel,
	}}
}

func tryGetIssueStatus(_ *pb.ProtocolID, key psm.StateKey) *pb.ProtocolStatus_Issue_ {
	credRep := e2.IssueCredRep.Try(psm.GetIssueCredRep(key))

	// TODO: save schema id parsed to db? copied from original implementation
	var credOfferMap map[string]interface{}
	dto.FromJSONStr(credRep.CredOffer, &credOfferMap)

	schemaID := credOfferMap["schema_id"].(string)

	attrs := make([]*pb.Protocol_Attribute, 0, len(credRep.Attributes))
	for _, credAttr := range credRep.Attributes {
		a := &pb.Protocol_Attribute{
			Name:  credAttr.Name,
			Value: credAttr.Value,
		}
		attrs = append(attrs, a)
	}
	return &pb.ProtocolStatus_Issue_{Issue: &pb.ProtocolStatus_Issue{
		CredDefID: credRep.CredDefID,
		SchemaID:  schemaID,
		Attrs:     attrs,
	}}
}

func tryGetProofStatus(_ *pb.ProtocolID, key psm.StateKey) *pb.ProtocolStatus_Proof {
	proofRep, err := psm.GetPresentProofRep(key)
	err2.Check(err)
	if proofRep == nil {
		return &pb.ProtocolStatus_Proof{Proof: &pb.Protocol_Proof{Attributes: nil}}
	}
	attrs := make([]*pb.Protocol_Proof_Attribute, 0, len(proofRep.Attributes))

	for _, attr := range proofRep.Attributes {
		a := &pb.Protocol_Proof_Attribute{
			Name:      attr.Name,
			CredDefID: attr.CredDefID,
			//Predicate: attr.Predicate,
		}
		attrs = append(attrs, a)
	}
	return &pb.ProtocolStatus_Proof{Proof: &pb.Protocol_Proof{Attributes: attrs}}
}

func tryGetTrustPingStatus(_ *pb.ProtocolID, _ psm.StateKey) *pb.ProtocolStatus_TrustPing_ {
	// todo: add TrustPingRep to DB if we need to track Replied status
	return &pb.ProtocolStatus_TrustPing_{TrustPing: &pb.ProtocolStatus_TrustPing{Replied: false}}
}

func tryGetBasicMessageStatus(_ *pb.ProtocolID, key psm.StateKey) *pb.ProtocolStatus_BasicMessage_ {
	msg, err := psm.GetBasicMessageRep(key)
	err2.Check(err)

	glog.V(1).Infoln("Get BasicMsg for:", key, "sent by me:", msg.SentByMe)
	return &pb.ProtocolStatus_BasicMessage_{BasicMessage: &pb.ProtocolStatus_BasicMessage{
		Content:       msg.Message,
		SentByMe:      msg.SentByMe,
		Delivered:     msg.Delivered,
		SentTimestamp: msg.SendTimestamp,
	}}
}

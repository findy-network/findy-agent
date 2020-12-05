package server

import (
	"context"

	pb "github.com/findy-network/findy-agent-api/grpc/agency"
	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-grpc/jwt"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

type didCommServer struct {
	pb.UnimplementedDIDCommServer
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

	glog.V(3).Infoln("run() call")

	ctx := e2.Ctx.Try(jwt.CheckTokenValidity(server.Context()))
	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	task := e2.Task.Try(taskFrom(protocol))
	glog.V(3).Infoln(caDID, "-agent starts protocol:", pb.Protocol_Type_name[int32(protocol.TypeId)])

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

func (s *didCommServer) Resume(ctx context.Context, state *pb.ProtocolState) (pid *pb.ProtocolID, err error) {
	defer err2.Return(&err)

	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	glog.V(1).Infoln(caDID, "-agent Resume protocol:", state.ProtocolId.TypeId, state.ProtocolId.Id)

	prot.Resume(receiver, uniqueTypeID(state.ProtocolId.Role, state.ProtocolId.TypeId),
		state.ProtocolId.Id, state.GetState() == pb.ProtocolState_ACK)

	return state.ProtocolId, nil
}

func (s *didCommServer) Release(ctx context.Context, id *pb.ProtocolID) (ps *pb.ProtocolID, err error) {
	defer err2.Return(&err)

	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	glog.V(1).Infoln(caDID, "-agent release protocol:", id.Id)
	key := psm.NewStateKey(receiver.WorkerEA(), id.Id)
	err2.Check(prot.AddFlagUpdatePSM(key, psm.Archiving))
	glog.V(1).Infoln(caDID, "-agent release OK", id.Id)

	return id, nil
}

func (s *didCommServer) Start(ctx context.Context, protocol *pb.Protocol) (pid *pb.ProtocolID, err error) {
	defer err2.Return(&err)

	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	task := e2.Task.Try(taskFrom(protocol))
	glog.V(1).Infoln(caDID, "-agent starts protocol:", pb.Protocol_Type_name[int32(protocol.TypeId)])
	prot.FindAndStartTask(receiver, task)
	return &pb.ProtocolID{Id: task.Nonce}, nil
}

func (s *didCommServer) Status(ctx context.Context, id *pb.ProtocolID) (ps *pb.ProtocolStatus, err error) {
	defer err2.Return(&err)

	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	key := psm.NewStateKey(receiver.WorkerEA(), id.Id)
	glog.V(1).Infoln(caDID, "-agent protocol status:", pb.Protocol_Type_name[int32(id.TypeId)], protocolName[id.TypeId])
	ps = tryProtocolStatus(id, key)
	return ps, nil
}

func tryProtocolStatus(id *pb.ProtocolID, key psm.StateKey) *pb.ProtocolStatus {
	statusJSON := dto.ToJSON(prot.GetStatus(protocolName[id.TypeId], &key))
	m := e2.PSM.Try(psm.GetPSM(key))
	state := &pb.ProtocolState{
		ProtocolId: id,
		State:      calcProtocolState(m),
	}
	if m != nil {
		state.ProtocolId.Role = roleType[m.Initiator]
	} else {
		glog.Warningf("cannot get protocol role for %s", key)
		state.ProtocolId.Role = pb.Protocol_UNKNOWN
	}
	ps := &pb.ProtocolStatus{
		State:      state,
		StatusJson: statusJSON,
	}
	switch id.TypeId {
	case pb.Protocol_CONNECT:
		ps.Status = tryGetConnectStatus(id, key)
	case pb.Protocol_ISSUE:
		ps.Status = tryGetIssueStatus(id, key)
	case pb.Protocol_PROOF:
		ps.Status = tryGetProofStatus(id, key)
	case pb.Protocol_TRUST_PING:
		ps.Status = tryGetTrustPingStatus(id, key)
	case pb.Protocol_BASIC_MESSAGE:
		ps.Status = tryGetBasicMessageStatus(id, key)

	}
	return ps
}

func calcProtocolState(m *psm.PSM) pb.ProtocolState_State {
	if m != nil {
		if m.PendingUserAction() {
			return pb.ProtocolState_WAIT_ACTION
		}
		if last := m.LastState(); last != nil {
			switch last.Sub.Pure() {
			case psm.Ready:
				if last.Sub&psm.ACK != 0 {
					return pb.ProtocolState_OK
				}
				return pb.ProtocolState_NACK
			case psm.Failure:
				return pb.ProtocolState_ERR
			}
		}
	}
	return pb.ProtocolState_RUNNING
}

func tryGetConnectStatus(_ *pb.ProtocolID, key psm.StateKey) *pb.ProtocolStatus_Connection_ {
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

	return &pb.ProtocolStatus_Connection_{Connection: &pb.ProtocolStatus_Connection{
		Id:            pw.Name,
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
		CredDefId: credRep.CredDefID,
		SchemaId:  schemaID,
		Attrs:     attrs,
	}}
}

func tryGetProofStatus(_ *pb.ProtocolID, key psm.StateKey) *pb.ProtocolStatus_Proof {
	proofRep, err := psm.GetPresentProofRep(key)
	err2.Check(err)
	if proofRep == nil {
		return &pb.ProtocolStatus_Proof{Proof: &pb.Protocol_Proof{Attrs: nil}}
	}
	attrs := make([]*pb.Protocol_Proof_Attr, 0, len(proofRep.Attributes))

	for _, attr := range proofRep.Attributes {
		a := &pb.Protocol_Proof_Attr{
			Name:      attr.Name,
			CredDefId: attr.CredDefID,
			Predicate: attr.Predicate,
		}
		attrs = append(attrs, a)
	}
	return &pb.ProtocolStatus_Proof{Proof: &pb.Protocol_Proof{Attrs: attrs}}
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

package server

import (
	"context"

	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/findy-network/findy-common-go/jwt"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

type didCommServer struct {
	pb.UnimplementedProtocolServiceServer
}

func (s *didCommServer) Run(
	protocol *pb.Protocol,
	server pb.ProtocolService_RunServer,
) (
	err error,
) {
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

	ctx := try.To1(jwt.CheckTokenValidity(server.Context()))
	caDID, receiver := try.To2(ca(ctx))

	task := try.To1(taskFrom(protocol))
	glog.V(3).Infoln(caDID, "-agent starts protocol:", protocol.TypeID)

	key := psm.NewStateKey(receiver.WorkerEA(), task.ID())
	statusChan := bus.WantAll.AddListener(key)
	userActionChan := bus.WantUserActions.AddListener(key)

	prot.FindAndStartTask(receiver, task)

	var statusCode pb.ProtocolState_State
loop:
	for {
		select {
		case status := <-statusChan:
			glog.V(3).Infof("grpc %s state in %s", status, task.ID())
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
					ProtocolID: &pb.ProtocolID{ID: task.ID()},
					State:      pb.ProtocolState_WAIT_ACTION,
				}
				try.To(server.Send(status))
			}
		}
	}
	glog.V(3).Infoln("out from grpc state:", statusCode)
	bus.WantAll.RmListener(key)
	bus.WantUserActions.RmListener(key)

	status := &pb.ProtocolState{
		ProtocolID: &pb.ProtocolID{ID: task.ID()},
		State:      statusCode,
	}
	try.To(server.Send(status))

	return nil
}

func (s *didCommServer) Resume(ctx context.Context, state *pb.ProtocolState) (pid *pb.ProtocolID, err error) {
	defer err2.Return(&err)

	caDID, receiver := try.To2(ca(ctx))
	glog.V(1).Infoln(caDID, "-agent Resume protocol:", state.ProtocolID.TypeID, state.ProtocolID.ID)

	prot.Resume(receiver, uniqueTypeID(state.ProtocolID.Role, state.ProtocolID.TypeID),
		state.ProtocolID.ID, state.GetState() == pb.ProtocolState_ACK)

	return state.ProtocolID, nil
}

func (s *didCommServer) Release(ctx context.Context, id *pb.ProtocolID) (ps *pb.ProtocolID, err error) {
	defer err2.Return(&err)

	caDID, receiver := try.To2(ca(ctx))
	glog.V(1).Infoln(caDID, "-agent release protocol:", id.ID)
	key := psm.NewStateKey(receiver.WorkerEA(), id.ID)
	try.To(prot.AddAndSetFlagUpdatePSM(key, psm.Archiving, 0))
	glog.V(1).Infoln(caDID, "-agent release OK", id.ID)

	return id, nil
}

func (s *didCommServer) Start(ctx context.Context, protocol *pb.Protocol) (pid *pb.ProtocolID, err error) {
	defer err2.Return(&err)

	caDID, receiver := try.To2(ca(ctx))
	task := try.To1(taskFrom(protocol))
	glog.V(1).Infoln(caDID, "-agent starts protocol:", protocol.TypeID)
	prot.FindAndStartTask(receiver, task)
	return &pb.ProtocolID{ID: task.ID()}, nil
}

func (s *didCommServer) Status(ctx context.Context, id *pb.ProtocolID) (ps *pb.ProtocolStatus, err error) {
	defer err2.Return(&err)

	caDID, receiver := try.To2(ca(ctx))
	key := psm.NewStateKey(receiver.WorkerEA(), id.ID)
	ps, _ = tryProtocolStatus(key)

	glog.V(1).Infoln(caDID, "-agent protocol status:", ps.State.ProtocolID.TypeID, protocolName[ps.State.ProtocolID.TypeID])
	return ps, nil
}

func tryProtocolStatus(key psm.StateKey) (ps *pb.ProtocolStatus, connID string) {
	m := try.To1(psm.GetPSM(key))
	protocolType := m.FirstState().T.ProtocolType()
	state := &pb.ProtocolState{
		ProtocolID: &pb.ProtocolID{
			ID:     key.Nonce,
			TypeID: protocolType,
		},
		State: calcProtocolState(m),
	}
	if m != nil {
		connID = m.ConnID
		state.ProtocolID.Role = m.Role
	} else {
		glog.Warningf("cannot get protocol role for %s", key)
		state.ProtocolID.Role = pb.Protocol_UNKNOWN
	}
	ps = &pb.ProtocolStatus{
		State: state,
	}
	// protocol implementors fill the status information
	ps = prot.FillStatus(protocolName[protocolType], key, ps)
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

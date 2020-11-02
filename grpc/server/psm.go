package server

import (
	"context"

	pb "github.com/findy-network/findy-agent-api/grpc/agency"
	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/mesg"
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

func (s *didCommServer) Unpause(ctx context.Context, state *pb.ProtocolState) (pid *pb.ProtocolID, err error) {
	defer err2.Return(&err)

	caDID, receiver := e2.StrRcvr.Try(ca(ctx))
	glog.V(1).Infoln(caDID, "-agent unpause protocol:", state.ProtocolId.TypeId, state.ProtocolId.Id)

	om := mesg.MsgCreator.Create(didcomm.MsgInit{
		Ready: state.GetState() == pb.ProtocolState_ACK,
		ID:    state.ProtocolId.Id,
	}).(didcomm.Msg)
	prot.Unpause(receiver, typeID[state.ProtocolId.TypeId], om)

	return state.ProtocolId, nil
}

func (s *didCommServer) Release(ctx context.Context, id *pb.ProtocolID) (ps *pb.ProtocolState, err error) {
	defer err2.Return(&err)

	panic("implement me")
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
	task := &comm.Task{
		Nonce:  id.Id,
		TypeID: typeID[id.TypeId],
	}
	key := psm.NewStateKey(receiver.WorkerEA(), task.Nonce)
	glog.V(1).Infoln(caDID, "-agent protocol status:", pb.Protocol_Type_name[int32(id.TypeId)])
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

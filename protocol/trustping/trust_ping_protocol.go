package trustping

import (
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

type taskTrustPing struct {
	comm.TaskBase
}

var trustPingProcessor = comm.ProtProc{
	Creator: createTrustPingTask,
	Starter: startTrustPing,
	Handlers: map[string]comm.HandlerFunc{
		pltype.HandlerPing:         handleTrustPing,
		pltype.HandlerPingResponse: handleTrustPingResponse,
	},
	FillStatus: fillTrustPingStatus,
}

func init() {
	gob.Register(&taskTrustPing{})
	prot.AddCreator(pltype.ProtocolTrustPing, trustPingProcessor)
	prot.AddStarter(pltype.CATrustPing, trustPingProcessor)
	prot.AddStatusProvider(pltype.ProtocolTrustPing, trustPingProcessor)
	comm.Proc.Add(pltype.ProtocolTrustPing, trustPingProcessor)
}

func createTrustPingTask(header *comm.TaskHeader, protocol *pb.Protocol) (t comm.Task, err error) {
	defer err2.Annotate("createTrustPingTask", &err)

	glog.V(1).Infof("Create task for TrustPing with connection id %s", header.ConnID)

	if protocol != nil && protocol.ConnectionID == "" {
		glog.Warningln("pinging first found connection, conn-id was empty")
	}

	return &taskTrustPing{
		TaskBase: comm.TaskBase{TaskHeader: *header},
	}, nil
}

func startTrustPing(ca comm.Receiver, t comm.Task) {
	defer err2.CatchTrace(func(err error) {
		glog.Error(err)
	})
	try.To(prot.StartPSM(prot.Initial{
		SendNext:    pltype.TrustPingPing,
		WaitingNext: pltype.TrustPingResponse,
		Ca:          ca,
		T:           t,
		Setup: func(key psm.StateKey, hdr didcomm.MessageHdr) error {
			return nil
		},
	}))
}

func handleTrustPing(packet comm.Packet) (err error) {
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.TrustPingResponse,
		WaitingNext: pltype.Terminate,
		InOut: func(connID string, im, om didcomm.MessageHdr) (ack bool, err error) {
			glog.V(3).Info("-- Thread ID: ", om.Thread().ID)
			return true, nil
		},
	})
}

func handleTrustPingResponse(packet comm.Packet) (err error) {
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.Terminate,
		WaitingNext: pltype.Terminate,
		InOut: func(connID string, im, om didcomm.MessageHdr) (ack bool, err error) {
			glog.V(3).Info("-- Thread ID: ", om.Thread().ID)
			return true, nil
		},
	})
}

func fillTrustPingStatus(workerDID string, taskID string, ps *pb.ProtocolStatus) *pb.ProtocolStatus {
	defer err2.CatchTrace(func(err error) {
		glog.Error("Failed to fill trust ping status: ", err)
	})

	assert.D.True(ps != nil)

	status := ps

	// TODO
	status.Status = &pb.ProtocolStatus_TrustPing{
		TrustPing: &pb.ProtocolStatus_TrustPingStatus{Replied: false},
	}

	return status
}

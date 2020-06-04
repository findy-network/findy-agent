package trustping

import (
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/optechlab/findy-agent/agent/comm"
	"github.com/optechlab/findy-agent/agent/didcomm"
	"github.com/optechlab/findy-agent/agent/pltype"
	"github.com/optechlab/findy-agent/agent/prot"
	"github.com/optechlab/findy-agent/agent/psm"
)

type statusTrustPing struct {
	Result string `json:"result"`
}

var trustPingProcessor = comm.ProtProc{
	Starter: startTrustPing,
	Handlers: map[string]comm.HandlerFunc{
		pltype.HandlerPing:         handleTrustPing,
		pltype.HandlerPingResponse: handleTrustPingResponse,
	},
	Status: getTrustPingStatus,
}

func init() {
	prot.AddStarter(pltype.CATrustPing, trustPingProcessor)
	prot.AddStatusProvider(pltype.ProtocolTrustPing, trustPingProcessor)
	comm.Proc.Add(pltype.ProtocolTrustPing, trustPingProcessor)
}

func startTrustPing(ca comm.Receiver, t *comm.Task) {
	defer err2.CatchTrace(func(err error) {
		glog.Error(err)
	})
	err2.Check(prot.StartPSM(prot.Initial{
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
		InOut: func(im, om didcomm.MessageHdr) (ack bool, err error) {
			glog.V(3).Info("-- Nonce: ", im.Thread().ID)
			return true, nil
		},
	})
}

func handleTrustPingResponse(packet comm.Packet) (err error) {
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.Terminate,
		WaitingNext: pltype.Terminate,
		InOut: func(im, om didcomm.MessageHdr) (ack bool, err error) {
			glog.V(3).Info("-- Nonce: ", im.Thread().ID)
			return true, nil
		},
	})
}

func getTrustPingStatus(workerDID string, taskID string) interface{} {
	// TODO:
	return statusTrustPing{
		Result: "NOT SUPPORTED",
	}
}

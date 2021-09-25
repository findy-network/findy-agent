package notification

import (
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/std/common"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

var processor = comm.ProtProc{Starter: startProtocol,
	Handlers: map[string]comm.HandlerFunc{
		pltype.HandlerProblemReport: handleProblemReport,
	}}

func init() {
	prot.AddStarter(pltype.CAProblemReport, processor)
	comm.Proc.Add(pltype.ProtocolNotification, processor)
}

func startProtocol(ca comm.Receiver, t *comm.Task) {
	defer err2.CatchTrace(func(err error) {
		glog.Error(err)
	})

	err2.Check(prot.StartPSM(prot.Initial{
		SendNext:    pltype.BasicMessageSend,
		WaitingNext: pltype.Terminate,
		Ca:          ca,
		T:           t,
		Setup: func(key psm.StateKey, om didcomm.MessageHdr) error {
			// todo: fill the report data here
			return nil
		},
	}))
}

func handleProblemReport(packet comm.Packet) (err error) {
	tHandler := func(connID string, im, om didcomm.MessageHdr) (ack bool, err error) {
		defer err2.Annotate("basic message", &err)

		problemReport := im.FieldObj().(*common.ProblemReport)

		glog.Info("Sent time:", problemReport.ExplainLongTxt)

		//key := psm.StateKey{
		//	DID:   packet.Receiver.Trans().MessagePipe().In.Did(),
		//	Nonce: im.Thread().ID,
		//}

		return true, nil
	}
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.Terminate,
		WaitingNext: pltype.Terminate,
		InOut:       tHandler,
	})
}

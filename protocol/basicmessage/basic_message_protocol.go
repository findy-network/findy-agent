package basicmessage

import (
	"encoding/gob"
	"time"

	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/std/basicmessage"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

type taskBasicMessage struct {
	comm.TaskBase
	Content string
}

// basicMessageProcessor is a protocol processor for Basic Message protocol.
var basicMessageProcessor = comm.ProtProc{
	Creator: createBasicMessageTask,
	Starter: startBasicMessage,
	Handlers: map[string]comm.HandlerFunc{
		pltype.HandlerMessage: handleBasicMessage,
	},
	FillStatus: fillBasicMessageStatus,
}

func init() {
	gob.Register(&taskBasicMessage{})
	prot.AddCreator(pltype.ProtocolBasicMessage, basicMessageProcessor)
	prot.AddStarter(pltype.CABasicMessage, basicMessageProcessor)
	prot.AddStatusProvider(pltype.ProtocolBasicMessage, basicMessageProcessor)
	comm.Proc.Add(pltype.ProtocolBasicMessage, basicMessageProcessor)
}

func createBasicMessageTask(header *comm.TaskHeader, protocol *pb.Protocol) (t comm.Task, err error) {
	defer err2.Annotate("createBasicMessageTask", &err)

	var content string
	if protocol != nil {
		assert.P.True(
			protocol.GetBasicMessage() != nil,
			"basic message protocol data missing")

		content = protocol.GetBasicMessage().GetContent()

		glog.V(1).Infof("Create task for BasicMessage with connection id %s", header.ConnID)
	}

	return &taskBasicMessage{
		TaskBase: comm.TaskBase{TaskHeader: *header},
		Content:  content,
	}, nil
}

func startBasicMessage(ca comm.Receiver, t comm.Task) {
	defer err2.CatchTrace(func(err error) {
		glog.Error(err)
	})

	try.To(prot.StartPSM(prot.Initial{
		SendNext:    pltype.BasicMessageSend,
		WaitingNext: pltype.Terminate,
		Ca:          ca,
		T:           t,
		Setup: func(key psm.StateKey, om didcomm.MessageHdr) (err error) {
			defer err2.Return(&err)

			bmTask, ok := t.(*taskBasicMessage)
			assert.P.True(ok)

			rep := &basicMessageRep{
				StateKey:  key,
				PwName:    bmTask.ConnectionID(),
				Message:   bmTask.Content,
				Timestamp: time.Now().UnixNano(),
				SentByMe:  true,
				Delivered: true,
			}
			try.To(psm.AddRep(rep))

			msg := om.FieldObj().(*basicmessage.Basicmessage)
			msg.Content = bmTask.Content
			return nil
		},
	}))
}

func handleBasicMessage(packet comm.Packet) (err error) {
	tHandler := func(connID string, im, om didcomm.MessageHdr) (ack bool, err error) {
		defer err2.Annotate("basic message", &err)

		pw, err := packet.Receiver.FindPWByDID(packet.Address.RcvrDID)
		try.To(err)
		assert.D.True(pw != nil, "pairwise is nil")

		bm := im.FieldObj().(*basicmessage.Basicmessage)

		if glog.V(3) {
			glog.Info("-- Thread id: ", im.Thread().ID)
			glog.Info("Basic msg from:", pw.Meta.Name)
			glog.Info("Sent time:", bm.SentTime)
			glog.Info("Content: ", bm.Content)
		}

		key := psm.StateKey{
			DID:   packet.Receiver.MyDID().Did(),
			Nonce: im.Thread().ID,
		}

		rep := &basicMessageRep{
			StateKey:      key,
			PwName:        pw.Meta.Name,
			Message:       bm.Content,
			SendTimestamp: bm.SentTime.Time.UnixNano(),
			Timestamp:     time.Now().UnixNano(),
			SentByMe:      false,
			Delivered:     true,
		}
		try.To(psm.AddRep(rep))

		return true, nil
	}
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.Terminate,
		WaitingNext: pltype.Terminate,
		InOut:       tHandler,
	})
}

func fillBasicMessageStatus(workerDID string, taskID string, ps *pb.ProtocolStatus) *pb.ProtocolStatus {
	defer err2.CatchTrace(func(err error) {
		glog.Error("Failed to fill basic message status: ", err)
	})

	assert.D.True(ps != nil)

	status := ps

	msg, err := getBasicMessageRep(workerDID, taskID)
	try.To(err)

	status.Status = &pb.ProtocolStatus_BasicMessage{BasicMessage: &pb.ProtocolStatus_BasicMessageStatus{
		Content:       msg.Message,
		SentByMe:      msg.SentByMe,
		Delivered:     msg.Delivered,
		SentTimestamp: msg.SendTimestamp,
	}}

	return status
}

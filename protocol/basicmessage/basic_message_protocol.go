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
)

type taskBasicMessage struct {
	comm.TaskBase
	Content string
}

type statusBasicMessage struct {
	PwName    string `json:"pairwise"`
	Message   string `json:"message"`
	SentByMe  bool   `json:"sentByMe"`
	Delivered bool   `json:"delivered"`
}

// basicMessageProcessor is a protocol processor for Basic Message protocol.
var basicMessageProcessor = comm.ProtProc{
	Creator: createBasicMessageTask,
	Starter: startBasicMessage,
	Handlers: map[string]comm.HandlerFunc{
		pltype.HandlerMessage: handleBasicMessage,
	},
	Status: getBasicMessageStatus,
}

func init() {
	gob.Register(&taskBasicMessage{})
	prot.AddCreator(pltype.CABasicMessage, basicMessageProcessor)
	prot.AddStarter(pltype.CABasicMessage, basicMessageProcessor)
	prot.AddStatusProvider(pltype.ProtocolBasicMessage, basicMessageProcessor)
	comm.Proc.Add(pltype.ProtocolBasicMessage, basicMessageProcessor)
}

func createBasicMessageTask(header *comm.TaskHeader, protocol *pb.Protocol) (t comm.Task, err error) {
	defer err2.Annotate("createBasicMessageTask", &err)

	assert.P.True(
		protocol.GetBasicMessage() != nil,
		"basic message protocol data missing")

	glog.V(1).Infof("Create task for BasicMessage with connection id %s", header.ConnID)

	return &taskBasicMessage{
		TaskBase: comm.TaskBase{TaskHeader: *header},
		Content:  protocol.GetBasicMessage().GetContent(),
	}, nil
}

func startBasicMessage(ca comm.Receiver, t comm.Task) {
	defer err2.CatchTrace(func(err error) {
		glog.Error(err)
	})

	err2.Check(prot.StartPSM(prot.Initial{
		SendNext:    pltype.BasicMessageSend,
		WaitingNext: pltype.Terminate,
		Ca:          ca,
		T:           t,
		Setup: func(key psm.StateKey, om didcomm.MessageHdr) (err error) {
			defer err2.Return(&err)

			bmTask, ok := t.(*taskBasicMessage)
			assert.P.True(ok)

			rep := &psm.BasicMessageRep{
				Key:       key,
				PwName:    bmTask.ConnectionID(),
				Message:   bmTask.Content,
				Timestamp: time.Now().UnixNano(),
				SentByMe:  true,
				Delivered: true,
			}
			err2.Check(psm.AddBasicMessageRep(rep))

			msg := om.FieldObj().(*basicmessage.Basicmessage)
			msg.Content = bmTask.Content
			return nil
		},
	}))
}

func handleBasicMessage(packet comm.Packet) (err error) {
	tHandler := func(connID string, im, om didcomm.MessageHdr) (ack bool, err error) {
		defer err2.Annotate("basic message", &err)

		_, name := err2.StrStr.Try(packet.Receiver.FindPW(packet.Address.RcvrDID))

		bm := im.FieldObj().(*basicmessage.Basicmessage)

		if glog.V(3) {
			glog.Info("-- Thread id: ", im.Thread().ID)
			glog.Info("Basic msg from:", name)
			glog.Info("Sent time:", bm.SentTime)
			glog.Info("Content: ", bm.Content)
		}

		key := psm.StateKey{
			DID:   packet.Receiver.Trans().MessagePipe().In.Did(),
			Nonce: im.Thread().ID,
		}

		rep := &psm.BasicMessageRep{
			Key:           key,
			PwName:        name,
			Message:       bm.Content,
			SendTimestamp: bm.SentTime.Time.UnixNano(),
			Timestamp:     time.Now().UnixNano(),
			SentByMe:      false,
			Delivered:     true,
		}
		err2.Check(psm.AddBasicMessageRep(rep))

		return true, nil
	}
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.Terminate,
		WaitingNext: pltype.Terminate,
		InOut:       tHandler,
	})
}

func getBasicMessageStatus(workerDID string, taskID string) interface{} {
	defer err2.CatchTrace(func(err error) {
		glog.Error("Failed to set basic message status: ", err)
	})
	key := &psm.StateKey{
		DID:   workerDID,
		Nonce: taskID,
	}
	msg, err := psm.GetBasicMessageRep(*key)
	err2.Check(err)

	return statusBasicMessage{
		PwName:    msg.PwName,
		Message:   msg.Message,
		Delivered: msg.Delivered, // TODO?
		SentByMe:  msg.SentByMe,
	}
}

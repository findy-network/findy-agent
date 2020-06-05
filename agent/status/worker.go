package status

import (
	"time"

	"github.com/findy-network/findy-agent/agent/cloud"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

type worker struct {
	ca, wa *cloud.Agent
}

type TaskParam struct {
	Type        string
	ID          string
	DeviceToken string
	TsSinceMs   *uint64
}

func NewWorker(a comm.Receiver) *worker {
	return &worker{ca: a.(*cloud.Agent)}
}

func (w *worker) Exec(t *TaskParam) interface{} {
	switch t.Type {
	case pltype.CATaskStatus:
		return w.taskStatus(t)
	case pltype.CATaskList:
		return w.listTasks(t)
	}
	return ""
}

func (w *worker) eaDID() string {
	meAddr := w.ca.CAEndp(true) // CA can give us w-EA's endpoint
	return meAddr.RcvrDID       // Get EA's DID, to build KeyState, etc.
}

func (w *worker) taskStatus(t *TaskParam) *prot.TaskStatus {
	return prot.StatusForTask(w.eaDID(), t.ID)
}

func (w *worker) TaskReady(t *TaskParam) (yes bool, err error) {
	key := psm.StateKey{
		DID:   w.eaDID(),
		Nonce: t.ID,
	}
	return psm.IsPSMReady(key)
}

func (w *worker) listTasks(t *TaskParam) *[]string {
	defer err2.CatchTrace(func(err error) {
		glog.Error("Error in status check: ", err)
	})

	if len(t.DeviceToken) > 0 {
		newDeviceID := &psm.DeviceIDRep{
			DID:         w.eaDID(),
			DeviceToken: t.DeviceToken,
		}
		err2.Check(psm.AddDeviceIDRep(newDeviceID))
	}

	var tsSince *int64
	if t.TsSinceMs != nil {
		tsSinceMs := int64(*t.TsSinceMs) * int64(time.Millisecond)
		tsSince = &tsSinceMs
	}

	psms, err := psm.AllPSM(w.eaDID(), tsSince)
	err2.Check(err)

	res := make([]string, 0)
	for i := 0; i < len(*psms); i++ {
		res = append(res, (*psms)[i].Key.Nonce)
	}
	return &res
}

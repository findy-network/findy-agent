package prot

import (
	"time"

	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

type TaskStatus struct {
	ID                string      `json:"id"`
	Type              string      `json:"type"`
	Status            string      `json:"status"`
	Name              string      `json:"name"`
	TimestampMs       uint64      `json:"timestamp"`
	PendingUserAction bool        `json:"pendingUserAction"`
	Payload           interface{} `json:"payload"`
}

const StatusReady = "ready"
const StatusWaiting = "waiting"

func StatusForTask(workerDID string, taskID string) *TaskStatus {
	defer err2.CatchTrace(func(err error) {
		glog.Error("Error in status check: ", err)
	})

	key := &psm.StateKey{
		DID:   workerDID,
		Nonce: taskID,
	}

	state := e2.PSM.Try(psm.GetPSM(*key))

	status := StatusReady
	if !state.IsReady() {
		status = StatusWaiting
	}

	pendingUserAction := state.PendingUserAction()
	taskStatus := &TaskStatus{
		ID:                taskID,
		Type:              state.Protocol(),
		Status:            status,
		Name:              state.PairwiseName(),
		TimestampMs:       uint64(state.Timestamp() / int64(time.Millisecond)),
		PendingUserAction: pendingUserAction,
	}

	if state.IsReady() || pendingUserAction {
		taskStatus.Payload = GetStatus(state.Protocol(), key)
	}

	return taskStatus
}

package psm

import (
	"encoding/gob"
	"fmt"

	"github.com/findy-network/findy-agent/agent/comm"
)

const (
	mockType       = "did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/basicmessage/1.0/message"
	mockStateDID   = "TEST"
	mockStateNonce = "1234"
)

// todo: this is copied here from agent package. Consider removing cycling
//  dependency by going black box testing which would allow using the original
//  function in the agent package.

// RegisterGobs the original version can be found from agent package.
func registerGobs() {
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
}

func testPSM(ts int64) *PSM {
	var states []State
	if ts != 0 {
		states = make([]State, 1)
		states[0] = State{
			PLInfo: PayloadInfo{
				Type: mockType,
			},
			Timestamp: ts,
			T: &comm.TaskBase{
				Head: comm.TaskHeader{
					ConnectionID: "pairwise",
				},
			},
		}
	}
	nonce := mockStateNonce
	if ts != 0 {
		nonce = fmt.Sprintf("%s%d", nonce, ts)
	}
	return &PSM{
		Key: StateKey{
			DID:   mockStateDID,
			Nonce: nonce,
		},
		InDID:  mockStateDID,
		States: states,
	}
}

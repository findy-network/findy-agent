package v1

import (
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/core"
	our "github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-agent/std/didexchange"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/lainio/err2/try"
)

var completeCreator = &completeFactor{}

type completeFactor struct{}

func (f *completeFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	c := &Complete{
		Type:   init.Type,
		ID:     init.AID,
		Thread: checkThread(&our.Thread{}, init.Nonce),
	}
	return newComplete(c)
}

func (f *completeFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return newCompleteMsg(data)
}

func init() {
	gob.Register(&completeImpl{})
	aries.Creator.Add(pltype.AriesDIDExchangeComplete, completeCreator)
	aries.Creator.Add(pltype.DIDOrgAriesDIDExchangeComplete, completeCreator)
}

func newComplete(c *Complete) (impl *completeImpl) {
	return &completeImpl{Complete: c}
}

func newCompleteMsg(data []byte) *completeImpl {
	var mImpl completeImpl
	dto.FromJSON(data, &mImpl)
	checkThread(mImpl.Complete.Thread, mImpl.Complete.Thread.PID)
	return &mImpl
}

type completeImpl struct {
	didexchange.UnsupportedPwMsgBase
	*Complete
}

func (m *completeImpl) FieldObj() interface{} {
	return m.Complete
}

func (m *completeImpl) JSON() []byte {
	return dto.ToJSONBytes(m.Complete)
}

func (m *completeImpl) Thread() *our.Thread {
	return m.Complete.Thread
}

func (m *completeImpl) ID() string {
	return m.Complete.ID
}

func (m *completeImpl) SetID(id string) {
	m.Complete.ID = id
}

func (m *completeImpl) Type() string {
	return m.Complete.Type
}

func (m *completeImpl) SetType(t string) {
	m.Complete.Type = t
}

func (m *completeImpl) Next(_ string, _ core.DID) (didcomm.Payload, psm.SubState, error) {
	// we are ready at this end for this protocol
	emptyMsg := aries.MsgCreator.Create(didcomm.MsgInit{})
	return aries.PayloadCreator.NewMsg(
		m.Complete.Thread.PID,
		pltype.DIDOrgAriesDIDExchangeComplete,
		emptyMsg,
	), psm.ReadyACK, nil
}

func (m *completeImpl) Wait() (didcomm.Payload, psm.SubState) {
	return try.To2(m.Next("", nil))
}

package didexchange1

import (
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	decorator0 "github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/golang/glog"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/protocol/decorator"
)

var CompleteCreator = &CompleteFactor{}

type CompleteFactor struct{}

func (f *CompleteFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	r := &Request{
		Type:   init.Type,
		ID:     init.AID,
		Thread: &decorator.Thread{ID: init.Nonce},
	}
	return NewRequest(nil, r)
}

func (f *CompleteFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return NewRequestMsg(data)
}

func init() {
	gob.Register(&CompleteImpl{})
	aries.Creator.Add(pltype.AriesDIDExchangeComplete, CompleteCreator)
	aries.Creator.Add(pltype.DIDOrgAriesDIDExchangeComplete, CompleteCreator)
}

func NewComplete(c *Complete) (impl *CompleteImpl) {
	return &CompleteImpl{c}
}

func NewCompleteMsg(data []byte) *CompleteImpl {
	var mImpl CompleteImpl
	dto.FromJSON(data, &mImpl)
	mImpl.checkThread()
	return &mImpl
}

type CompleteImpl struct {
	*Complete
}

func (m *CompleteImpl) FieldObj() interface{} {
	return m.Complete
}

func (m *CompleteImpl) JSON() []byte {
	return dto.ToJSONBytes(m.Complete)
}

func (m *CompleteImpl) checkThread() {
	legacyThread := decorator0.CheckThread(m.Thread(), m.Complete.ID)
	m.Complete.Thread = &decorator.Thread{
		ID:             legacyThread.ID,
		PID:            legacyThread.PID,
		SenderOrder:    legacyThread.SenderOrder,
		ReceivedOrders: legacyThread.ReceivedOrders,
	}
}

func (m *CompleteImpl) Thread() *decorator0.Thread {
	return &decorator0.Thread{
		ID:             m.Complete.Thread.ID,
		PID:            m.Complete.Thread.PID,
		SenderOrder:    m.Complete.Thread.SenderOrder,
		ReceivedOrders: m.Complete.Thread.ReceivedOrders,
	}
}

func (m *CompleteImpl) ID() string {
	return m.Complete.ID
}

func (m *CompleteImpl) SetID(id string) {
	m.Complete.ID = id
}

func (m *CompleteImpl) Type() string {
	return m.Complete.Type
}

func (m *CompleteImpl) SetType(t string) {
	m.Complete.Type = t
}

func (m *CompleteImpl) Nonce() string {
	if th := m.Complete.Thread; th != nil {
		return th.ID
	}
	glog.Warning("Returning ID() for nonce/thread_id")
	return m.ID()
}

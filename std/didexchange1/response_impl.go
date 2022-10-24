package didexchange1

import (
	"encoding/gob"
	"encoding/json"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/core"
	decorator0 "github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/golang/glog"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/protocol/decorator"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

var ResponseCreator = &ResponseFactor{}

type ResponseFactor struct{}

type ResponseImpl struct {
	commonImpl
	*Response
}

func (f *ResponseFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	r := &Response{
		Type:   init.Type,
		ID:     init.AID,
		Thread: &decorator.Thread{ID: init.Nonce},
	}
	return NewResponse(nil, r)
}

func (f *ResponseFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return NewResponseMsg(data)
}

func init() {
	gob.Register(&ResponseImpl{})
	aries.Creator.Add(pltype.AriesDIDExchangeResponse, ResponseCreator)
	aries.Creator.Add(pltype.DIDOrgAriesDIDExchangeResponse, ResponseCreator)
}

func NewResponse(doc core.DIDDoc, r *Response) (resp *ResponseImpl) {
	defer err2.Catch(func(err error) {
		glog.Error("failed to marshal diddoc when creating V1 response")
	})
	if doc != nil {
		r.DIDDoc = newDIDDocAttach(utils.UUID(), try.To1(json.Marshal(doc)))
	}
	return &ResponseImpl{commonImpl{
		commonData{
			DID:    r.DID,
			DIDDoc: r.DIDDoc,
		},
	}, r}
}

func NewResponseMsg(data []byte) *ResponseImpl {
	var mImpl ResponseImpl
	dto.FromJSON(data, &mImpl)
	mImpl.checkThread()
	mImpl.commonData = commonData{
		DID:    mImpl.Response.DID,
		DIDDoc: mImpl.Response.DIDDoc,
	}
	return &mImpl
}

func (m *ResponseImpl) FieldObj() interface{} {
	return m.Response
}

func (m *ResponseImpl) JSON() []byte {
	return dto.ToJSONBytes(m.Response)
}

func (m *ResponseImpl) Name() string {
	panic("unsupported")
}

func (m *ResponseImpl) checkThread() {
	legacyThread := decorator0.CheckThread(m.Thread(), m.Response.ID)
	m.Response.Thread = &decorator.Thread{
		ID:             legacyThread.ID,
		PID:            legacyThread.PID,
		SenderOrder:    legacyThread.SenderOrder,
		ReceivedOrders: legacyThread.ReceivedOrders,
	}
}

func (m *ResponseImpl) Thread() *decorator0.Thread {
	return &decorator0.Thread{
		ID:             m.Response.Thread.ID,
		PID:            m.Response.Thread.PID,
		SenderOrder:    m.Response.Thread.SenderOrder,
		ReceivedOrders: m.Response.Thread.ReceivedOrders,
	}
}

func (m *ResponseImpl) ID() string {
	return m.Response.ID
}

func (m *ResponseImpl) SetID(id string) {
	m.Response.ID = id
}

func (m *ResponseImpl) Type() string {
	return m.Response.Type
}

func (m *ResponseImpl) SetType(t string) {
	m.Response.Type = t
}

func (m *ResponseImpl) Nonce() string {
	if th := m.Response.Thread; th != nil {
		return th.ID
	}
	glog.Warning("Returning ID() for nonce/thread_id")
	return m.ID()
}

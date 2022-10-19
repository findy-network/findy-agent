package didexchange1

import (
	"encoding/gob"
	"encoding/json"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

var RequestCreator = &RequestFactor{}

type RequestFactor struct{}

func (f *RequestFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	r := &Request{
		Type:   init.Type,
		ID:     init.AID,
		Thread: &decorator.Thread{ID: init.Nonce},
	}
	return NewRequest(nil, r)
}

func (f *RequestFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return NewRequestMsg(data)
}

func init() {
	gob.Register(&RequestImpl{})
	aries.Creator.Add(pltype.AriesDIDExchangeRequest, RequestCreator)
	aries.Creator.Add(pltype.DIDOrgAriesDIDExchangeRequest, RequestCreator)
}

func NewRequest(doc core.DIDDoc, r *Request) (req *RequestImpl) {
	defer err2.Catch(func(err error) {
		glog.Error("failed to marshal diddoc when creating V1 request")
	})
	if doc != nil {
		r.DIDDoc = newDIDDocAttach(utils.UUID(), try.To1(json.Marshal(doc)))
	}
	return &RequestImpl{commonImpl{
		commonData{
			DID:    r.DID,
			DIDDoc: r.DIDDoc,
		},
	}, r}
}

func NewRequestMsg(data []byte) *RequestImpl {
	var mImpl RequestImpl
	dto.FromJSON(data, &mImpl)
	mImpl.checkThread()
	mImpl.commonData = commonData{
		DID:    mImpl.Request.DID,
		DIDDoc: mImpl.Request.DIDDoc,
	}
	return &mImpl
}

type RequestImpl struct {
	commonImpl
	*Request
}

func (m *RequestImpl) FieldObj() interface{} {
	return m.Request
}

func (m *RequestImpl) JSON() []byte {
	return dto.ToJSONBytes(m.Request)
}

func (m *RequestImpl) Name() string { // Todo: names should be Label
	return m.Label
}

func (m *RequestImpl) checkThread() {
	m.Request.Thread = decorator.CheckThread(m.Request.Thread, m.Request.ID)
}

func (m *RequestImpl) Thread() *decorator.Thread {
	return m.Request.Thread
}

func (m *RequestImpl) ID() string {
	return m.Request.ID
}

func (m *RequestImpl) SetID(id string) {
	m.Request.ID = id
}

func (m *RequestImpl) Type() string {
	return m.Request.Type
}

func (m *RequestImpl) SetType(t string) {
	m.Request.Type = t
}

func (m *RequestImpl) Nonce() string {
	if th := m.Request.Thread; th != nil {
		return th.ID
	}
	glog.Warning("Returning ID() for nonce/thread_id")
	return m.ID()
}

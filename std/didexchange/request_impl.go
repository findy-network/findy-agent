package didexchange

import (
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/std/common"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/golang/glog"
	"github.com/lainio/err2/assert"
	"github.com/mr-tron/base58"
)

var Creator = &Factor{}

type Factor struct{}

func (f *Factor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	r := &Request{
		Type:   init.Type,
		ID:     init.AID,
		Thread: &decorator.Thread{ID: init.Nonce},
	}
	return NewRequest(r)
}

func (f *Factor) NewMessage(data []byte) didcomm.MessageHdr {
	return NewMsg(data)
}

func init() {
	gob.Register(&RequestImpl{})
	aries.Creator.Add(pltype.AriesConnectionRequest, Creator)
	aries.Creator.Add(pltype.DIDOrgAriesConnectionRequest, Creator)
}

func NewRequest(r *Request) *RequestImpl {
	return &RequestImpl{Request: r}
}

func NewMsg(data []byte) *RequestImpl {
	var mImpl RequestImpl
	dto.FromJSON(data, &mImpl)
	mImpl.checkThread()
	return &mImpl
}

func (m *RequestImpl) checkThread() {
	m.Request.Thread = decorator.CheckThread(m.Request.Thread, m.Request.ID)
}

type RequestImpl struct {
	*Request
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

func (m *RequestImpl) JSON() []byte {
	return dto.ToJSONBytes(m)
}

func (m *RequestImpl) FieldObj() interface{} {
	return m.Request
}

func (m *RequestImpl) Nonce() string {
	if th := m.Request.Thread; th != nil {
		return th.ID
	}
	glog.Warning("Returning ID() for nonce/thread_id")
	return m.ID()
}

func (m *RequestImpl) Name() string { // Todo: names should be Label
	return m.Label
}

func (m *RequestImpl) Did() string {
	return m.Connection.DID
}

func (m *RequestImpl) VerKey() string {
	vm := common.VM(m.Connection.DIDDoc, 0)
	return base58.Encode(vm.Value)
}

func (m *RequestImpl) Endpoint() service.Addr {
	assert.NotNil(m)
	assert.NotNil(m.Connection)
	assert.That(m.Connection.DIDDoc != nil)

	if len(common.Services(m.Connection.DIDDoc)) == 0 {
		return service.Addr{}
	}

	serv := common.Service(m.Connection.DIDDoc, 0)
	addr := serv.ServiceEndpoint
	key := serv.RecipientKeys[0]

	return service.Addr{Endp: addr, Key: key}
}

func (m *RequestImpl) SetEndpoint(ae service.Addr) {
	panic("todo: we should not be here.. at least with the current impl")
}

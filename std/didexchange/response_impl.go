package didexchange

import (
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/golang/glog"
	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	"github.com/mr-tron/base58"
)

var ResponseCreator = &ResponseFactor{}

type ResponseFactor struct{}

func (f *ResponseFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	r := &Response{
		Type:   init.Type,
		ID:     init.AID,
		Thread: &decorator.Thread{ID: init.Nonce},
	}
	return NewResponse(r)
}

func (f *ResponseFactor) Create(init didcomm.MsgInit) didcomm.MessageHdr {
	var doc *did.Doc
	DID := init.Did

	if init.DIDObj != nil {
		ep := service.Addr{Endp: init.Endpoint, Key: init.EndpVerKey}
		doc = init.DIDObj.NewDoc(ep).(*did.Doc)
		DID = init.DIDObj.Did()
	}

	resImpl := &ResponseImpl{Response: &Response{
		Type:   init.Type,
		ID:     init.ID,
		Thread: &decorator.Thread{ID: init.Nonce},
		Connection: &Connection{
			DID:    DID,
			DIDDoc: doc,
		},
	}}
	return resImpl
}

func (f *ResponseFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return NewResponseMsg(data)
}

func init() {
	gob.Register(&ResponseImpl{})
	aries.Creator.Add(pltype.AriesConnectionResponse, ResponseCreator)
	aries.Creator.Add(pltype.DIDOrgAriesConnectionResponse, ResponseCreator)
}

func NewResponse(r *Response) *ResponseImpl {
	return &ResponseImpl{Response: r}
}

func NewResponseMsg(data []byte) *ResponseImpl {
	var mImpl ResponseImpl
	dto.FromJSON(data, &mImpl)
	mImpl.checkThread()
	return &mImpl
}

func (m *ResponseImpl) checkThread() {
	m.Response.Thread = decorator.CheckThread(m.Response.Thread, m.Response.ID)
}

type ResponseImpl struct {
	*Response
}

func (m *ResponseImpl) Thread() *decorator.Thread {
	return m.Response.Thread
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

func (m *ResponseImpl) JSON() []byte {
	//m.Response.Sign()
	return dto.ToJSONBytes(m)
}

func (m *ResponseImpl) FieldObj() interface{} {
	return m.Response
}

func (m *ResponseImpl) Nonce() string {
	if th := m.Response.Thread; th != nil {
		return th.ID
	}
	glog.Warning("returning ID() for nonce/thread_id")
	return m.ID()
}

func (m *ResponseImpl) Did() string {
	return m.Connection.DID
}

func (m *ResponseImpl) VerKey() string {
	if len(m.Connection.DIDDoc.VerificationMethod) == 0 {
		return ""
	}
	return base58.Encode(m.Connection.DIDDoc.VerificationMethod[0].Value)
	//return m.Connection.Doc.VerificationMethod[0].PublicKeyBase58
}

func (m *ResponseImpl) Name() string { // Todo: names should be Label
	return ""
}

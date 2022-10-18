package didexchange1

import (
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"strings"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-agent/std/common"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-agent/std/sov/did"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
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
	return NewRequest(nil, r)
}

func (f *Factor) NewMessage(data []byte) didcomm.MessageHdr {
	return NewMsg(data)
}

func init() {
	gob.Register(&RequestImpl{})
	aries.Creator.Add(pltype.AriesDIDExchangeRequest, Creator)
	aries.Creator.Add(pltype.DIDOrgAriesDIDExchangeRequest, Creator)
}

func NewRequest(doc core.DIDDoc, r *Request) (req *RequestImpl) {
	defer err2.Catch(func(err error) {
		glog.Error("failed to marshal diddoc when creating V1 request")
	})
	if doc != nil {
		r.DIDDoc = newDIDDocAttach(utils.UUID(), try.To1(json.Marshal(doc)))
	}
	return &RequestImpl{Request: r}
}

func newDIDDocAttach(id string, didDoc []byte) []decorator.Attachment {
	data := decorator.AttachmentData{
		Base64: base64.StdEncoding.EncodeToString(didDoc)}
	attachment := []decorator.Attachment{{
		//ID:       id,
		MimeType: "application/json",
		Data:     data,
	}}
	return attachment
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
	glog.V(11).Infoln("Did() is returning:", m.DID)
	rawDID := strings.TrimPrefix(m.DID, "did:sov:")
	if rawDID != m.DID {
		glog.V(3).Infoln("+++ normalizing Did()", m.DID, " ==>", rawDID)
	}
	return rawDID
}

func (m *RequestImpl) DIDDocument() (coreDoc core.DIDDoc, err error) {
	defer err2.Returnf(&err, "request DID doc")
	assert.That(m.DIDDoc != nil)

	var doc did.Doc
	didDocBytes := try.To1(base64.StdEncoding.DecodeString(m.DIDDoc[0].Data.Base64))
	try.To(json.Unmarshal(didDocBytes, &doc))
	return &doc, nil
}

func (m *RequestImpl) VerKey() (key string, err error) {
	defer err2.Returnf(&err, "request VerKey")
	doc := try.To1(m.DIDDocument())
	vm := common.VM(doc, 0)
	return base58.Encode(vm.Value), nil
}

func (m *RequestImpl) Endpoint() (addr service.Addr, err error) {
	defer err2.Returnf(&err, "request Endpoint")

	assert.NotNil(m)
	doc := try.To1(m.DIDDocument())

	if len(common.Services(doc)) == 0 {
		return service.Addr{}, nil
	}

	serv := common.Service(doc, 0)
	endp := serv.ServiceEndpoint
	key := serv.RecipientKeys[0]

	return service.Addr{Endp: endp, Key: key}, nil
}

func (m *RequestImpl) SetEndpoint(ae service.Addr) {
	panic("todo: we should not be here.. at least with the current impl")
}

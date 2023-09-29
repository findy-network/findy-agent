package v0

import (
	"encoding/gob"
	"strings"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-agent/std/common"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
	"github.com/mr-tron/base58"
)

var requestCreator = &requestFactor{}

type requestFactor struct{}

func (f *requestFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	r := &Request{
		Type:   init.Type,
		ID:     init.AID,
		Thread: &decorator.Thread{ID: init.Nonce},
	}
	return newRequest(r)
}

func (f *requestFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return newRequestMsg(data)
}

func init() {
	gob.Register(&requestImpl{})
	aries.Creator.Add(pltype.AriesConnectionRequest, requestCreator)
	aries.Creator.Add(pltype.DIDOrgAriesConnectionRequest, requestCreator)
}

func newRequest(r *Request) *requestImpl {
	return &requestImpl{Request: r}
}

func newRequestMsg(data []byte) *requestImpl {
	var mImpl requestImpl
	dto.FromJSON(data, &mImpl)
	mImpl.checkThread()
	return &mImpl
}

func (m *requestImpl) checkThread() {
	m.Request.Thread = decorator.CheckThread(m.Request.Thread, m.Request.ID)
}

type requestImpl struct {
	*Request
}

func (m *requestImpl) Thread() *decorator.Thread {
	return m.Request.Thread
}

func (m *requestImpl) ID() string {
	return m.Request.ID
}

func (m *requestImpl) SetID(id string) {
	m.Request.ID = id
}

func (m *requestImpl) Type() string {
	return m.Request.Type
}

func (m *requestImpl) SetType(t string) {
	m.Request.Type = t
}

func (m *requestImpl) JSON() []byte {
	return dto.ToJSONBytes(m)
}

func (m *requestImpl) FieldObj() interface{} {
	return m.Request
}

func (m *requestImpl) Label() string {
	return m.Request.Label
}

func (m *requestImpl) Did() string {
	glog.V(11).Infoln("Did() is returning:", m.Connection.DID)
	rawDID := strings.TrimPrefix(m.Connection.DID, "did:sov:")
	if rawDID != m.Connection.DID {
		glog.V(3).Infoln("+++ normalizing Did()", m.Connection.DID, " ==>", rawDID)
	}
	return rawDID
}

func (m *requestImpl) VerKey() string {
	vm := common.VM(m.Connection.DIDDoc, 0)
	return base58.Encode(vm.Value)
}

func (m *requestImpl) Endpoint() service.Addr {
	defer err2.Catch(err2.Err(func(err error) {
		glog.Errorf("Getting endpoint failed: %s", err)
	}))

	assert.NotNil(m)
	assert.NotNil(m.Connection)
	assert.That(m.Connection.DIDDoc != nil)

	if len(common.Services(m.Connection.DIDDoc)) == 0 {
		return service.Addr{}
	}

	serv := common.Service(m.Connection.DIDDoc, 0)
	addr := try.To1(serv.ServiceEndpoint.URI())
	key := serv.RecipientKeys[0] // TODO: convert did:key

	return service.Addr{Endp: addr, Key: key}
}

func (m *requestImpl) DIDDocument() core.DIDDoc {
	return m.Connection.DIDDoc
}

func (m *requestImpl) RoutingKeys() []string {
	return common.Service(m.Connection.DIDDoc, 0).RoutingKeys
}

func (m *requestImpl) Verify(_ core.DID) error {
	return nil
}

func (m *requestImpl) PayloadToSend(_ string, ourDID core.DID) (pl didcomm.Payload, st psm.SubState, err error) {
	defer err2.Handle(&err, "next for v0 request")
	endp := try.To1(ourDID.AEndp())
	msg := try.To1(newResponse(&Response{
		Connection: &Connection{
			DID:    ourDID.Did(),
			DIDDoc: ourDID.NewDoc(endp),
		},
		Thread: &decorator.Thread{ID: m.Request.Thread.ID},
	}, ourDID))
	return aries.PayloadCreator.NewMsg(m.Request.Thread.ID, pltype.AriesConnectionResponse, msg), psm.Sending, nil
}

func (m *requestImpl) PayloadToWait() (didcomm.Payload, psm.SubState) {
	return aries.PayloadCreator.New(
		didcomm.PayloadInit{
			ID:   m.Request.Thread.ID,
			Type: pltype.AriesConnectionResponse,
		}), psm.Waiting
}

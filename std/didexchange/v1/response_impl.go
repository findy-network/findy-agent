package v1

import (
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/core"
	our "github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

var responseCreator = &responseFactor{}

type responseFactor struct{}

type responseImpl struct {
	commonImpl
	*Response
}

func (f *responseFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	return &responseImpl{
		Response: &Response{
			Type:   init.Type,
			ID:     init.AID,
			Thread: checkThread(&our.Thread{}, init.Nonce),
		},
	}
}

func (f *responseFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return newResponseMsg(data)
}

func init() {
	gob.Register(&responseImpl{})
	aries.Creator.Add(pltype.AriesDIDExchangeResponse, responseCreator)
	aries.Creator.Add(pltype.DIDOrgAriesDIDExchangeResponse, responseCreator)
}

func newResponse(ourDID core.DID, r *Response) (resp *responseImpl, err error) {
	defer err2.Returnf(&err, "new response %s", ourDID.Did())

	r.DIDDoc = try.To1(newDIDDocAttach(ourDID))
	return &responseImpl{commonImpl{
		commonData{
			DID:    r.DID,
			DIDDoc: r.DIDDoc,
		},
	}, r}, nil
}

func newResponseMsg(data []byte) *responseImpl {
	var mImpl responseImpl
	dto.FromJSON(data, &mImpl)
	checkThread(mImpl.Response.Thread, mImpl.Response.Thread.PID)
	mImpl.commonData = commonData{
		DID:    mImpl.Response.DID,
		DIDDoc: mImpl.Response.DIDDoc,
	}
	return &mImpl
}

func (m *responseImpl) FieldObj() interface{} {
	return m.Response
}

func (m *responseImpl) JSON() []byte {
	return dto.ToJSONBytes(m.Response)
}

func (m *responseImpl) Label() string {
	panic("unsupported")
}

func (m *responseImpl) Thread() *our.Thread {
	return m.Response.Thread
}

func (m *responseImpl) ID() string {
	return m.Response.ID
}

func (m *responseImpl) SetID(id string) {
	m.Response.ID = id
}

func (m *responseImpl) Type() string {
	return m.Response.Type
}

func (m *responseImpl) SetType(t string) {
	m.Response.Type = t
}

func (m *responseImpl) PayloadToSend(_ string, _ core.DID) (didcomm.Payload, psm.SubState, error) {
	msg := newComplete(&Complete{
		Thread: checkThread(&our.Thread{}, m.Response.Thread.PID),
	})
	return aries.PayloadCreator.NewMsg(
		m.Response.Thread.PID,
		pltype.DIDOrgAriesDIDExchangeComplete,
		msg,
	), psm.Sending, nil
}

func (m *responseImpl) PayloadToWait() (didcomm.Payload, psm.SubState) {
	emptyMsg := aries.MsgCreator.Create(didcomm.MsgInit{})
	return aries.PayloadCreator.NewMsg(
		m.Response.Thread.PID,
		pltype.DIDOrgAriesDIDExchangeComplete,
		emptyMsg,
	), psm.ReadyACK
}

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

var requestCreator = &requestFactor{}

type requestFactor struct{}

func (f *requestFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	return &requestImpl{
		Request: &Request{
			Type:   init.Type,
			ID:     init.AID,
			Thread: checkThread(&our.Thread{}, init.Nonce),
		},
	}
}

func (f *requestFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return newRequestMsg(data)
}

func init() {
	gob.Register(&requestImpl{})
	aries.Creator.Add(pltype.AriesDIDExchangeRequest, requestCreator)
	aries.Creator.Add(pltype.DIDOrgAriesDIDExchangeRequest, requestCreator)
}

func newRequest(ourDID core.DID, r *Request) (req *requestImpl, err error) {
	defer err2.Returnf(&err, "new v1 request")
	r.DIDDoc = try.To1(newDIDDocAttach(ourDID))
	return &requestImpl{commonImpl{
		commonData{
			DID:    r.DID,
			DIDDoc: r.DIDDoc,
		},
	}, r}, nil
}

func newRequestMsg(data []byte) *requestImpl {
	var mImpl requestImpl
	dto.FromJSON(data, &mImpl)
	checkThread(mImpl.Request.Thread, mImpl.Request.Thread.PID)
	mImpl.commonData = commonData{
		DID:    mImpl.Request.DID,
		DIDDoc: mImpl.Request.DIDDoc,
	}
	return &mImpl
}

type requestImpl struct {
	commonImpl
	*Request
}

func (m *requestImpl) FieldObj() interface{} {
	return m.Request
}

func (m *requestImpl) JSON() []byte {
	return dto.ToJSONBytes(m.Request)
}

func (m *requestImpl) Label() string {
	return m.Request.Label
}

func (m *requestImpl) Thread() *our.Thread {
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

func (m *requestImpl) Next(_ string, ourDID core.DID) (pl didcomm.Payload, st psm.SubState, err error) {
	defer err2.Returnf(&err, "next for v1 request")
	msg := try.To1(newResponse(ourDID, &Response{
		DID:    ourDID.Did(),
		Thread: checkThread(&our.Thread{}, m.Request.Thread.PID),
	}))

	return aries.PayloadCreator.NewMsg(m.Request.Thread.PID, pltype.DIDOrgAriesDIDExchangeResponse, msg), psm.Sending, nil
}

func (m *requestImpl) Wait() (didcomm.Payload, psm.SubState) {
	return aries.PayloadCreator.New(
		didcomm.PayloadInit{
			ID:   m.Request.Thread.PID,
			Type: pltype.DIDOrgAriesDIDExchangeResponse,
		}), psm.Waiting
}

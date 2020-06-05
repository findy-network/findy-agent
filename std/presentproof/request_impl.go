package presentproof

import (
	"encoding/base64"
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-wrapper-go/dto"
)

var RequestCreator = &RequestFactor{}

type RequestFactor struct{}

func (f *RequestFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	m := &Request{
		Type:    init.Type,
		ID:      init.AID,
		Comment: init.Info,
		Thread:  decorator.CheckThread(init.Thread, init.AID),
	}
	return NewRequest(m)
}

func (f *RequestFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return NewRequestMsg(data)
}

func init() {
	gob.Register(&RequestImpl{})
	aries.Creator.Add(pltype.PresentProofRequest, RequestCreator)
}

func NewRequest(r *Request) *RequestImpl {
	return &RequestImpl{Request: r}
}

func NewRequestMsg(data []byte) *RequestImpl {
	var mImpl RequestImpl
	dto.FromJSON(data, &mImpl)
	mImpl.checkThread()
	return &mImpl
}

// MARK: Helper functions

func ProofReq(req *Request) (pr map[string]interface{}, err error) {
	data, e := ProofReqData(req)
	if e != nil {
		return nil, e
	}

	proofReq := make(map[string]interface{})
	dto.FromJSON(data, &proofReq)
	return proofReq, nil
}

func ProofReqData(req *Request) (data []byte, err error) {
	return base64.StdEncoding.DecodeString(req.RequestPresentations[0].Data.Base64)
}

func (p *RequestImpl) checkThread() {
	p.Request.Thread = decorator.CheckThread(p.Request.Thread, p.Request.ID)
}

type RequestImpl struct {
	*Request
}

func NewRequestPresentation(ID string, proofReq []byte) []decorator.Attachment {
	data := decorator.AttachmentData{Base64: base64.StdEncoding.EncodeToString(proofReq)}
	rp := []decorator.Attachment{{
		ID:       ID,
		MimeType: "application/json",
		Data:     data,
	}}
	return rp
}

func (p *RequestImpl) ID() string {
	return p.Request.ID
}

func (p *RequestImpl) Type() string {
	return p.Request.Type
}

func (p *RequestImpl) SetID(id string) {
	p.Request.ID = id
}

func (p *RequestImpl) SetType(t string) {
	p.Request.Type = t
}

func (p *RequestImpl) JSON() []byte {
	return dto.ToJSONBytes(p)
}

func (p *RequestImpl) Thread() *decorator.Thread {
	//if p.Request.Thread == nil {}
	return p.Request.Thread
}

func (p *RequestImpl) FieldObj() interface{} {
	return p.Request
}

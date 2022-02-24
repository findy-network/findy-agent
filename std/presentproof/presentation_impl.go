package presentproof

import (
	"encoding/base64"
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-common-go/dto"
)

var PresentationCreator = &PresentationFactor{}

type PresentationFactor struct{}

func (f *PresentationFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	m := &Presentation{
		Type:    init.Type,
		ID:      init.AID,
		Comment: init.Info,
		Thread:  decorator.CheckThread(init.Thread, init.AID),
	}
	return NewPresentation(m)
}

func (f *PresentationFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return NewPresentationMsg(data)
}

func init() {
	gob.Register(&PresentationImpl{})
	aries.Creator.Add(pltype.PresentProofPresentation, PresentationCreator)
	aries.Creator.Add(pltype.DIDOrgPresentProofPresentation, PresentationCreator)
}

func NewPresentation(r *Presentation) *PresentationImpl {
	return &PresentationImpl{Presentation: r}
}

func NewPresentationMsg(data []byte) *PresentationImpl {
	var mImpl PresentationImpl
	dto.FromJSON(data, &mImpl)
	mImpl.checkThread()
	return &mImpl
}

// MARK: Helpers

func Proof(p *Presentation) (data []byte, err error) {
	return base64.StdEncoding.DecodeString(p.PresentationAttaches[0].Data.Base64)
}

func (p *PresentationImpl) checkThread() {
	p.Presentation.Thread = decorator.CheckThread(p.Presentation.Thread, p.Presentation.ID)
}

// MARK: Struct
type PresentationImpl struct {
	*Presentation
}

func NewPresentationAttach(ID string, proofReq []byte) []decorator.Attachment {
	data := decorator.AttachmentData{Base64: base64.StdEncoding.EncodeToString(proofReq)}
	rp := []decorator.Attachment{{
		ID:       ID,
		MimeType: "application/json",
		Data:     data,
	}}
	return rp
}

func (p *PresentationImpl) ID() string {
	return p.Presentation.ID
}

func (p *PresentationImpl) Type() string {
	return p.Presentation.Type
}

func (p *PresentationImpl) SetID(id string) {
	p.Presentation.ID = id
}

func (p *PresentationImpl) SetType(t string) {
	p.Presentation.Type = t
}

func (p *PresentationImpl) JSON() []byte {
	return dto.ToJSONBytes(p)
}

func (p *PresentationImpl) Thread() *decorator.Thread {
	//if p.Presentation.Thread == nil {}
	return p.Presentation.Thread
}

func (p *PresentationImpl) FieldObj() interface{} {
	return p.Presentation
}

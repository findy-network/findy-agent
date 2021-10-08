package issuecredential

import (
	"encoding/base64"
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-wrapper-go/dto"
)

var OfferCreator = &OfferFactor{}

type OfferFactor struct{}

func (f *OfferFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	m := &Offer{
		Type:    init.Type,
		ID:      init.AID,
		Comment: init.Info,
		Thread:  decorator.CheckThread(init.Thread, init.AID),
	}
	return NewOffer(m)
}

func (f *OfferFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return NewOfferMsg(data)
}

func init() {
	gob.Register(&OfferImpl{})
	aries.Creator.Add(pltype.IssueCredentialOffer, OfferCreator)
	aries.Creator.Add(pltype.DIDOrgIssueCredentialOffer, OfferCreator)
}

func NewOffer(r *Offer) *OfferImpl {
	return &OfferImpl{Offer: r}
}

func NewOfferMsg(data []byte) *OfferImpl {
	var mImpl OfferImpl
	dto.FromJSON(data, &mImpl)
	mImpl.checkThread()
	return &mImpl
}

// MARK: Helpers

func OfferAttach(p *Offer) (data []byte, err error) {
	return base64.StdEncoding.DecodeString(p.OffersAttach[0].Data.Base64)
}

func (p *OfferImpl) checkThread() {
	p.Offer.Thread = decorator.CheckThread(p.Offer.Thread, p.Offer.ID)
}

// MARK: Struct
type OfferImpl struct {
	*Offer
}

func NewOfferAttach(offer []byte) []decorator.Attachment {
	data := decorator.AttachmentData{
		Base64: base64.StdEncoding.EncodeToString(offer)}
	rp := []decorator.Attachment{{
		ID:       "libindy-cred-offer-0",
		MimeType: "application/json",
		Data:     data,
	}}
	return rp
}

func (p *OfferImpl) ID() string {
	return p.Offer.ID
}

func (p *OfferImpl) Type() string {
	return p.Offer.Type
}

func (p *OfferImpl) SetID(id string) {
	p.Offer.ID = id
}

func (p *OfferImpl) SetType(t string) {
	p.Offer.Type = t
}

func (p *OfferImpl) JSON() []byte {
	return dto.ToJSONBytes(p)
}

func (p *OfferImpl) Thread() *decorator.Thread {
	//if p.Offer.Thread == nil {}
	return p.Offer.Thread
}

func (p *OfferImpl) FieldObj() interface{} {
	return p.Offer
}

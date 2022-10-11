package presentproof

import (
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-common-go/dto"
)

var Creator = &Factor{}

type Factor struct{}

func (f *Factor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	propose := &Propose{
		Type:    init.Type,
		ID:      init.AID,
		Comment: init.Info,
		Thread:  decorator.CheckThread(init.Thread, init.AID),
	}
	return NewPropose(propose)
}

func (f *Factor) NewMessage(data []byte) didcomm.MessageHdr {
	return NewMsg(data)
}

func init() {
	gob.Register(&ProposeImpl{})
	aries.Creator.Add(pltype.PresentProofPropose, Creator)
	aries.Creator.Add(pltype.DIDOrgPresentProofPropose, Creator)
}

func NewPropose(r *Propose) *ProposeImpl {
	return &ProposeImpl{Propose: r}
}

func NewMsg(data []byte) *ProposeImpl {
	var mImpl ProposeImpl
	dto.FromJSON(data, &mImpl)
	mImpl.checkThread()
	return &mImpl
}

func (p *ProposeImpl) checkThread() {
	p.Propose.Thread = decorator.CheckThread(p.Propose.Thread, p.Propose.ID)
}

type ProposeImpl struct {
	*Propose
}

// todo: do we really need ID here already? currently only used in tests, so the
//
//	ID isn't relevant at the moment
func newPropose(ID, credDefID string, values []string) *ProposeImpl {
	prev := NewPreview(values, credDefID)

	prop := &Propose{
		Type:                 pltype.PresentProofPropose,
		ID:                   ID,
		Comment:              "",
		PresentationProposal: prev,
		Thread:               decorator.NewThread(ID, ""),
	}
	return NewPropose(prop)
}

// todo: currently only used with the previous and in tests! This will be important later.
func NewPreview(values []string, credDefID string) *Preview {
	attrs := make([]Attribute, len(values))

	for i, value := range values {
		a := Attribute{
			Name:      value,
			CredDefID: credDefID,
		}
		attrs[i] = a
	}
	prev := &Preview{
		Type:       pltype.PresentationPreviewObj,
		Attributes: attrs,
		Predicates: nil,
	}
	return prev
}

func NewPreviewWithAttributes(attrs []Attribute) *Preview {
	prev := &Preview{
		Type:       pltype.PresentationPreviewObj,
		Attributes: attrs,
		Predicates: make([]Predicate, 0),
	}
	return prev
}

func (p *ProposeImpl) ID() string {
	return p.Propose.ID
}

func (p *ProposeImpl) Type() string {
	return p.Propose.Type
}

func (p *ProposeImpl) SetID(id string) {
	p.Propose.ID = id
}

func (p *ProposeImpl) SetType(t string) {
	p.Propose.Type = t
}

func (p *ProposeImpl) JSON() []byte {
	return dto.ToJSONBytes(p)
}

func (p *ProposeImpl) Thread() *decorator.Thread {
	//if p.Propose.Thread == nil {}
	return p.Propose.Thread
}

func (p *ProposeImpl) FieldObj() interface{} {
	return p.Propose
}

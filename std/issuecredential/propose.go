package issuecredential

import (
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
	"github.com/findy-network/findy-wrapper-go/dto"
)

var ProposeCreator = &ProposeFactor{}

type ProposeFactor struct{}

func (f *ProposeFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	m := &Propose{
		Type:    init.Type,
		ID:      init.AID,
		Comment: init.Info,
		Thread:  decorator.CheckThread(init.Thread, init.AID),
	}
	return NewPropose(m)
}

func (f *ProposeFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return NewProposeMsg(data)
}

func init() {
	gob.Register(&ProposeImpl{})
	aries.Creator.Add(pltype.IssueCredentialPropose, ProposeCreator)
	aries.Creator.Add(pltype.DIDOrgIssueCredentialPropose, ProposeCreator)
}

func NewPropose(r *Propose) *ProposeImpl {
	return &ProposeImpl{Propose: r}
}

func NewProposeMsg(data []byte) *ProposeImpl {
	var mImpl ProposeImpl
	dto.FromJSON(data, &mImpl)
	mImpl.checkThread()
	return &mImpl
}

// MARK: Helpers

// NewPreviewCredentialRaw creates a new PreviewCredential from JSON array which
// includes anoncreds.CredDefAttrs. NOTE! this is obsolete: for legacy protocol.
func NewPreviewCredentialRaw(values string) PreviewCredential {
	type attrT map[string]anoncreds.CredDefAttr

	array := make(attrT)
	dto.FromJSONStr(values, &array)
	var attrs = make([]Attribute, len(array))

	i := 0
	for key, value := range array {
		a := Attribute{Name: key, Value: value.Raw}
		attrs[i] = a
		i++
	}

	return PreviewCredential{
		Type:       pltype.IssueCredentialCredentialPreview,
		Attributes: attrs,
	}
}

// NewPreviewCredential creates a new PreviewCredential from JSON array which
// includes Attributes as Name Value pairs.
func NewPreviewCredential(values string) PreviewCredential {
	var attrs = make([]Attribute, 0, 4)
	dto.FromJSONStr(values, &attrs)

	return PreviewCredential{
		Type:       pltype.IssueCredentialCredentialPreview,
		Attributes: attrs,
	}
}

func PreviewCredentialToValues(credential PreviewCredential) string {
	attrs := make([]Attribute, len(credential.Attributes))

	for i, str := range credential.Attributes {
		// take mime type away
		a := Attribute{Name: str.Name, Value: str.Value}
		attrs[i] = a
	}
	return dto.ToJSON(attrs)
}

func PreviewCredentialToCodedValues(credential PreviewCredential) string {
	type attrT map[string]anoncreds.CredDefAttr

	rMap := make(attrT)

	for _, attr := range credential.Attributes {
		a := anoncreds.CredDefAttr{}
		a.SetRawAries(attr.Value)
		rMap[attr.Name] = a
	}
	return dto.ToJSON(rMap)
}

func (p *ProposeImpl) checkThread() {
	p.Propose.Thread = decorator.CheckThread(p.Propose.Thread, p.Propose.ID)
}

// MARK: Struct
type ProposeImpl struct {
	*Propose
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

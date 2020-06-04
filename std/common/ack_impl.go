package common

import (
	"encoding/gob"

	"github.com/optechlab/findy-agent/agent/aries"
	"github.com/optechlab/findy-agent/agent/didcomm"
	"github.com/optechlab/findy-agent/agent/pltype"
	"github.com/optechlab/findy-agent/std/decorator"
	"github.com/optechlab/findy-go/dto"
)

var AckCreator = &AckFactor{}

type AckFactor struct{}

func (f *AckFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	m := &Ack{
		Type:   init.Type,
		ID:     init.AID,
		Status: init.Info,
		Thread: decorator.CheckThread(init.Thread, init.AID),
	}
	return NewAck(m)
}

func (f *AckFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return NewAckMsg(data)
}

func init() {
	gob.Register(&AckImpl{})
	aries.Creator.Add(pltype.IssueCredentialACK, AckCreator)
	aries.Creator.Add(pltype.PresentProofACK, AckCreator)
}

func NewAck(r *Ack) *AckImpl {
	return &AckImpl{Ack: r}
}

func NewAckMsg(data []byte) *AckImpl {
	var mImpl AckImpl
	dto.FromJSON(data, &mImpl)
	mImpl.checkThread()
	return &mImpl
}

// MARK: Helpers

func (p *AckImpl) checkThread() {
	p.Ack.Thread = decorator.CheckThread(p.Ack.Thread, p.Ack.ID)
}

// MARK: Struct
type AckImpl struct {
	*Ack
}

func (p *AckImpl) ID() string {
	return p.Ack.ID
}

func (p *AckImpl) Type() string {
	return p.Ack.Type
}

func (p *AckImpl) SetID(id string) {
	p.Ack.ID = id
}

func (p *AckImpl) SetType(t string) {
	p.Ack.Type = t
}

func (p *AckImpl) JSON() []byte {
	return dto.ToJSONBytes(p)
}

func (p *AckImpl) Thread() *decorator.Thread {
	//if p.Ack.Thread == nil {}
	return p.Ack.Thread
}

func (p *AckImpl) FieldObj() interface{} {
	return p.Ack
}

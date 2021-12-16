package common

import (
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/lainio/err2/assert"
)

var ForwardCreator = &ForwardFactor{}

type ForwardFactor struct{}

func (f *ForwardFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	m := &Forward{
		Type: init.Type,
		ID:   init.AID,
		To:   init.To,
		Msg:  init.MsgBytes,
	}
	return NewForward(m)
}

func (f *ForwardFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return NewForwardMsg(data)
}

func init() {
	gob.Register(&ForwardImpl{})
	aries.Creator.Add(pltype.RoutingForward, ForwardCreator)
	aries.Creator.Add(pltype.DIDOrgRoutingForward, ForwardCreator)
}

func NewForward(r *Forward) *ForwardImpl {
	return &ForwardImpl{Forward: r}
}

func NewForwardMsg(data []byte) *ForwardImpl {
	var mImpl ForwardImpl
	dto.FromJSON(data, &mImpl)
	return &mImpl
}

// MARK: Struct
type ForwardImpl struct {
	*Forward
}

func (p *ForwardImpl) ID() string {
	return p.Forward.ID
}

func (p *ForwardImpl) Type() string {
	return p.Forward.Type
}

func (p *ForwardImpl) SetID(id string) {
	p.Forward.ID = id
}

func (p *ForwardImpl) SetType(t string) {
	p.Forward.Type = t
}

func (p *ForwardImpl) JSON() []byte {
	return dto.ToJSONBytes(p)
}

func (p *ForwardImpl) Thread() *decorator.Thread {
	assert.D.True(false, "Should not be here")
	return nil
}

func (p *ForwardImpl) FieldObj() interface{} {
	return p.Forward
}

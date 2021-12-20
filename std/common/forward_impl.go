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

var forwardCreator = &forwardFactor{}

type forwardFactor struct{}

func (f *forwardFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	m := &Forward{
		Type: init.Type,
		ID:   init.AID,
		To:   init.To,
		Msg:  init.Msg,
	}
	return NewForward(m)
}

func (f *forwardFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return NewForwardMsg(data)
}

func init() {
	gob.Register(&forwardImpl{})
	aries.Creator.Add(pltype.RoutingForward, forwardCreator)
	aries.Creator.Add(pltype.DIDOrgRoutingForward, forwardCreator)
}

func NewForward(r *Forward) *forwardImpl {
	return &forwardImpl{Forward: r}
}

func NewForwardMsg(data []byte) *forwardImpl {
	var mImpl forwardImpl
	dto.FromJSON(data, &mImpl)
	return &mImpl
}

// MARK: Struct
type forwardImpl struct {
	*Forward
}

func (p *forwardImpl) ID() string {
	return p.Forward.ID
}

func (p *forwardImpl) Type() string {
	return p.Forward.Type
}

func (p *forwardImpl) SetID(id string) {
	p.Forward.ID = id
}

func (p *forwardImpl) SetType(t string) {
	p.Forward.Type = t
}

func (p *forwardImpl) JSON() []byte {
	return dto.ToJSONBytes(p)
}

func (p *forwardImpl) Thread() *decorator.Thread {
	assert.D.True(false, "Should not be here")
	return nil
}

func (p *forwardImpl) FieldObj() interface{} {
	return p.Forward
}

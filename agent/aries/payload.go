/*
Package aries is implementation package for didcomm messages. See related
package mesg which is for our legacy messages. Both aries and mesg messages
share same interfaces defined in the didcomm package. didcomm defines a message
factor interfaces as well. With the help of the factoring system, the actual
messages can be constructed from the incoming messages with the correct type. We
use statically typed JSON messages i.e. they are always mapped to corresponding
Go struct.
*/
package aries

import (
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-common-go/dto"
)

var PayloadCreator = PayloadFactor{}

func init() {
	gob.Register(&PayloadImpl{})
	didcomm.CreatorGod.AddPayloadCreator(pltype.Aries, PayloadCreator)
	didcomm.CreatorGod.AddPayloadCreator(pltype.DIDOrgAries, PayloadCreator)
}

var Creator = &Factor{factors: make(map[string]didcomm.Factor)}

type Factor struct {
	factors map[string]didcomm.Factor
}

func (f *Factor) Add(t string, factor didcomm.Factor) {
	f.factors[t] = factor
}

type PayloadFactor struct{}

// NewFromData creates a new Aries PL in correct Go struct type. If @Type is
// associated to Go struct type which is registered to this Factor, it's used.
// If not a generic type is used.
func (f PayloadFactor) NewFromData(data []byte) didcomm.Payload {
	pl := &PayloadImpl{MessageHdr: newMsg(data)}
	t, id := pl.Type(), pl.ID()

	factor, ok := Creator.factors[pl.Type()]
	if !ok {
		return pl
	}
	m := factor.NewMessage(data)
	return f.NewMsg(id, t, m)
}

// New creates a new Aries PL with PayloadInit struct. The type of the Msg is
// generic.
func (f PayloadFactor) New(pi didcomm.PayloadInit) didcomm.Payload {
	pi.MsgInit.Type = pi.Type
	pi.MsgInit.AID = pi.AID

	msg := MsgCreator.Create(pi.MsgInit)
	return &PayloadImpl{MessageHdr: msg}
}

// NewMsg creates a new PL by ID, Type and already created internal Msg.
func (f PayloadFactor) NewMsg(id, t string, m didcomm.MessageHdr) didcomm.Payload {
	m.SetType(t)
	m.SetID(id)
	return &PayloadImpl{MessageHdr: m}
}

type PayloadImpl struct {
	didcomm.MessageHdr
}

func (pl *PayloadImpl) MsgHdr() didcomm.MessageHdr {
	return pl.MessageHdr
}

func (pl *PayloadImpl) ThreadID() string {
	if th := pl.Thread(); th != nil {
		if th.PID != "" {
			return th.PID
		}
		if th.ID != "" {
			return th.ID
		}
	}
	return pl.ID()
}

func (pl *PayloadImpl) SetThread(t *decorator.Thread) {
	panic("no implementation")
	//pl.Msg.(*MsgImpl).Thread = t
}

func (pl *PayloadImpl) Creator() didcomm.PayloadFactor {
	return didcomm.CreatorGod.PayloadCreator(pl.Namespace())
}

func (pl *PayloadImpl) MsgCreator() didcomm.MsgFactor {
	return didcomm.CreatorGod.MsgCreator(pl.Namespace())
}

func (pl *PayloadImpl) Data() []byte {
	return dto.ToGOB(pl)
}

func (pl *PayloadImpl) FieldObj() interface{} {
	return pl.MessageHdr
}

func (pl *PayloadImpl) Message() didcomm.Msg {
	return pl.MessageHdr.(didcomm.Msg)
}

func (pl *PayloadImpl) ID() string {
	return pl.MessageHdr.ID()
}

func (pl *PayloadImpl) Type() string {
	return pl.MessageHdr.Type()
}

func (pl *PayloadImpl) Protocol() string {
	return didcomm.FieldAtInd(pl.Type(), 1)
}

func (pl *PayloadImpl) ProtocolMsg() string {
	return didcomm.FieldAtInd(pl.Type(), 3)
}

func (pl *PayloadImpl) Namespace() string {
	return didcomm.FieldAtInd(pl.Type(), 0)
}

func ProtocolForType(typeStr string) string {
	return didcomm.FieldAtInd(typeStr, 1)
}

func ProtocolMsgForType(typeStr string) string {
	return didcomm.FieldAtInd(typeStr, 3)
}

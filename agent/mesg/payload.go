/*
Package mesg is implementation package for didcomm messages. See related
package aries which is for our legacy messages. Both aries and mesg messages
share same interfaces defined in the didcomm package. didcomm defines a message
factor interfaces as well. With the help of the factoring system, the actual
messages can be constructed from the incoming messages with the correct type. We
use statically typed JSON messages i.e. they are always mapped to corresponding
Go struct.

Because the development was started with indy's first agent to agent protocol,
and because the message types were taken from it, we had to refactor message
types when we implemented aries protocol. With the current didcomm package based
messaging system we can use polymorphic approach to all of the didcomm messages.
We doesn't need to know if the message is in coming indy message or aries
message.
*/
package mesg

import (
	"encoding/gob"
	"encoding/json"

	"github.com/golang/glog"
	"github.com/optechlab/findy-agent/agent/didcomm"
	"github.com/optechlab/findy-agent/agent/pltype"
	"github.com/optechlab/findy-agent/std/decorator"
)

var PayloadCreator = PayloadFactor{}

func init() {
	gob.Register(&PayloadImpl{})
	didcomm.CreatorGod.AddPayloadCreator(pltype.Agent, PayloadCreator)
	didcomm.CreatorGod.AddPayloadCreator(pltype.CA, PayloadCreator)
	didcomm.CreatorGod.AddPayloadCreator(pltype.SA, PayloadCreator)
}

type PayloadFactor struct{}

func (f PayloadFactor) NewFromData(data []byte) didcomm.Payload {
	return &PayloadImpl{Payload: NewPayload(data)}
}

func (f PayloadFactor) New(pi didcomm.PayloadInit) didcomm.Payload {
	msg := CreateMsg(pi.MsgInit)
	return &PayloadImpl{Payload: &Payload{ID: pi.ID, Type: pi.Type, Message: msg}}
}

func (f PayloadFactor) NewMsg(id, t string, m didcomm.MessageHdr) didcomm.Payload {
	msg := m.FieldObj().(*Msg)
	return &PayloadImpl{Payload: &Payload{ID: id, Type: t, Message: *msg}}
}

func (f PayloadFactor) NewError(pl didcomm.Payload, err error) didcomm.Payload {
	msg := pl.Message().FieldObj().(*Msg)
	msg.Error = err.Error()
	return &PayloadImpl{Payload: &Payload{ID: pl.ID(), Type: pl.Type(), Message: *msg}}
}

type PayloadImpl struct {
	*Payload
}

func (pl *PayloadImpl) ThreadID() string {
	return pl.ID()
}

func (pl *PayloadImpl) Thread() *decorator.Thread {
	panic("not in this api")
}

func (pl *PayloadImpl) SetThread(t *decorator.Thread) {
	panic("not in this api")
}

func (pl *PayloadImpl) Creator() didcomm.PayloadFactor {
	return didcomm.CreatorGod.PayloadCreator(pl.Namespace())
}

func (pl *PayloadImpl) MsgCreator() didcomm.MsgFactor {
	return didcomm.CreatorGod.MsgCreator(pl.Namespace())
}

func (pl *PayloadImpl) SetType(t string) {
	pl.Payload.Type = t
}

func (pl *PayloadImpl) Type() string {
	return pl.Payload.Type
}

func NewPayloadImpl(pl *Payload) didcomm.Payload {
	return &PayloadImpl{Payload: pl}
}

func NewPayloadBase(id, t string) didcomm.Payload {
	return &PayloadImpl{Payload: &Payload{ID: id, Type: t, Message: Msg{}}}
}

func NewPayloadWithMsg(id, t string, m *Msg) didcomm.Payload {
	return &PayloadImpl{Payload: &Payload{ID: id, Type: t, Message: *m}}
}

func (pl *PayloadImpl) FieldObj() interface{} {
	return pl.Payload
}

func (pl *PayloadImpl) Message() didcomm.Msg {
	return &MsgImpl{&pl.Payload.Message}
}

func (pl *PayloadImpl) MsgHdr() didcomm.MessageHdr {
	return &MsgImpl{&pl.Payload.Message}
}

func (pl *PayloadImpl) ID() string {
	return pl.Payload.ID
}

type Payload struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Message Msg    `json:"message"`
}

func (pl *Payload) JSON() []byte {
	output, err := json.Marshal(pl)
	if err != nil {
		glog.Error("Error marshalling to JSON:", err)
		return nil
	}
	return output
}

func NewPayload(data []byte) (p *Payload) {
	p = new(Payload)
	err := json.Unmarshal(data, p)
	if err != nil {
		glog.Error("Error marshalling from JSON: ", err.Error())
		return nil
	}
	return
}

func (pl *Payload) Protocol() string {
	return ProtocolForType(pl.Type)
}

func (pl *Payload) ProtocolMsg() string {
	return ProtocolMsgForType(pl.Type)
}

func (pl *Payload) Namespace() string {
	return didcomm.FieldAtInd(pl.Type, 0)
}

func ProtocolForType(typeStr string) string {
	return didcomm.FieldAtInd(typeStr, 1)
}

func ProtocolMsgForType(typeStr string) string {
	return didcomm.FieldAtInd(typeStr, 3)
}

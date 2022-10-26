package aries

import (
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-common-go/dto"
)

var MsgCreator = MsgFactor{}

func init() {
	gob.Register(&msgImpl{})
	didcomm.CreatorGod.AddMsgCreator(pltype.Aries, MsgCreator)
	didcomm.CreatorGod.AddMsgCreator(pltype.DIDOrgAries, MsgCreator)
}

type MsgFactor struct{}

func (f MsgFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	m := createMsg(init)
	return &msgImpl{Msg: &m}
}

func (f MsgFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return newMsg(data)
}

func (f MsgFactor) Create(d didcomm.MsgInit) didcomm.MessageHdr {
	factor, ok := Creator.factors[d.Type]
	if !ok {
		m := createMsg(d)
		return &msgImpl{Msg: &m}
	}
	return factor.NewMsg(d)
}

func createMsg(d didcomm.MsgInit) Msg {
	th := d.Thread
	if th == nil {
		th = decorator.NewThread(d.Nonce, "")
	}
	m := Msg{
		AID:    d.AID,
		Type:   d.Type,
		Thread: th,
		ID:     d.ID,
		Ready:  d.Ready,
		Msg:    d.Msg,
	}
	return m
}

type msgImpl struct {
	*Msg
}

func (m *msgImpl) Thread() *decorator.Thread {
	return m.Msg.Thread
}

func (m *msgImpl) ID() string {
	return m.Msg.AID
}

func (m *msgImpl) SetID(id string) {
	m.Msg.AID = id
}

func (m *msgImpl) Type() string {
	return m.Msg.Type
}

func (m *msgImpl) SetType(t string) {
	m.Msg.Type = t
}

func (m *msgImpl) JSON() []byte {
	return dto.ToJSONBytes(m.Msg)
}

func (m *msgImpl) SubLevelID() string {
	return m.Msg.ID
}

func (m *msgImpl) Ready() bool {
	return m.Msg.Ready
}

func (m *msgImpl) FieldObj() interface{} {
	return m.Msg
}

type Msg struct {
	Type string `json:"@type,omitempty"`
	AID  string `json:"@id,omitempty"`

	Thread *decorator.Thread `json:"~thread,omitempty"`

	ID    string                 `json:"id,omitempty"`    // Used for transferring additional ID like the Cred Def ID
	Ready bool                   `json:"ready,omitempty"` // In queries tells if something is ready when true
	Msg   map[string]interface{} `json:"msg,omitempty"`   // Forwarded message
}

func newMsg(data []byte) *msgImpl {
	var mImpl msgImpl
	dto.FromJSON(data, &mImpl)
	return &mImpl
}

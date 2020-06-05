package basicmessage

import (
	"encoding/gob"
	"time"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-wrapper-go/dto"
)

var Creator = &Factor{}

type Factor struct{}

func (f *Factor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	m := &Basicmessage{
		Type:     init.Type,
		ID:       init.AID,
		Content:  init.Info,
		SentTime: AriesTime{Time: time.Now()},
		Thread:   decorator.CheckThread(init.Thread, init.AID),
		//SentTime: AriesTime(time.Now()),
	}
	return NewBasicmessage(m)
}

func (f *Factor) NewMessage(data []byte) didcomm.MessageHdr {
	return NewBasicmessageMsg(data)
}

func init() {
	gob.Register(&Impl{})
	aries.Creator.Add(pltype.BasicMessageSend, Creator)
}

func NewBasicmessage(r *Basicmessage) *Impl {
	return &Impl{Basicmessage: r}
}

func NewBasicmessageMsg(data []byte) *Impl {
	var mImpl Impl
	dto.FromJSON(data, &mImpl)
	mImpl.checkThread()
	return &mImpl
}

// MARK: Helpers

func (p *Impl) checkThread() {
	p.Basicmessage.Thread = decorator.CheckThread(p.Basicmessage.Thread, p.Basicmessage.ID)
}

// MARK: Struct
type Impl struct {
	*Basicmessage
}

func (p *Impl) ID() string {
	return p.Basicmessage.ID
}

func (p *Impl) Type() string {
	return p.Basicmessage.Type
}

func (p *Impl) SetID(id string) {
	p.Basicmessage.ID = id
}

func (p *Impl) SetType(t string) {
	p.Basicmessage.Type = t
}

func (p *Impl) JSON() []byte {
	return dto.ToJSONBytes(p)
}

func (p *Impl) Thread() *decorator.Thread {
	//if p.Basicmessage.Thread == nil {}
	return p.Basicmessage.Thread
}

func (p *Impl) FieldObj() interface{} {
	return p.Basicmessage
}

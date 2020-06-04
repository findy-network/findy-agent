package txp

import (
	"github.com/optechlab/findy-agent/agent/didcomm"
	"github.com/optechlab/findy-agent/agent/mesg"
	"github.com/optechlab/findy-agent/agent/sec"
	"github.com/optechlab/findy-agent/agent/ssi"
)

// Transport is communication mechanism between agents
type Transport struct {
	PLPipe  sec.Pipe // Payload communication pipe
	MsgPipe sec.Pipe // Message communication
	Endp    string   // Given endpoint, Note! this is for the version 1 structure
}

func (t Transport) Payload() sec.Pipe {
	return t.PLPipe
}

func (t Transport) Message() sec.Pipe {
	return t.MsgPipe
}

func (t Transport) Endpoint() string {
	return t.Endp
}

func (t Transport) SetMessageOut(d *ssi.DID) {
	t.MsgPipe.Out = d
}

func (t Transport) EncMsg(msg *mesg.Msg) *mesg.Msg {
	return msg.Encrypt(t.MsgPipe)
}

func (t Transport) DecMsg(msg *mesg.Msg) *mesg.Msg {
	return msg.Decrypt(t.MsgPipe)
}

type Trans interface {
	PayloadPipe() sec.Pipe
	MessagePipe() sec.Pipe

	SetMessageOut(d *ssi.DID)

	EncMsg(msg *mesg.Msg) *mesg.Msg
	DecMsg(msg *mesg.Msg) *mesg.Msg

	EncDIDComMsg(msg didcomm.Msg) didcomm.Msg
	DecDIDComMsg(msg didcomm.Msg) didcomm.Msg

	Call(msgType string, msg *mesg.Msg) (rp *mesg.Payload, err error)
	DIDComCallEndp(endp, msgType string, msg didcomm.Msg) (rp didcomm.Payload, err error)
	EndpAddr() string
	String() string
}

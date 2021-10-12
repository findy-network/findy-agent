package txp

import (
	"github.com/findy-network/findy-agent/agent/sec"
	"github.com/findy-network/findy-agent/agent/ssi"
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

type Trans interface {
	PayloadPipe() sec.Pipe
	MessagePipe() sec.Pipe

	SetMessageOut(d *ssi.DID)

	EndpAddr() string
	String() string
}

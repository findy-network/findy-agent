package comm

import (
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/endp"
)

type Packet struct {
	Payload  didcomm.Payload
	Address  *endp.Addr
	Receiver Receiver
}

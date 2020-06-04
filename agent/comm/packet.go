package comm

import (
	"github.com/optechlab/findy-agent/agent/didcomm"
	"github.com/optechlab/findy-agent/agent/endp"
)

type Packet struct {
	Payload  didcomm.Payload
	Address  *endp.Addr
	Receiver Receiver
}

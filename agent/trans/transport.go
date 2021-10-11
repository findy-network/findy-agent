package trans

import (
	"fmt"
	"strconv"

	"github.com/findy-network/findy-agent/agent/sec"
	"github.com/findy-network/findy-agent/agent/ssi"
)

// Transport is communication mechanism between EA to CA, client to server.
// Server side is not supported yet, but coming. With Transport EA can
// communicate easily with its CA.
type Transport struct {
	PLPipe  sec.Pipe // Payload communication pipe
	MsgPipe sec.Pipe // Message communication
	Endp    string   // Given endpoint
}

func (tr Transport) String() string {
	inStr := tr.PLPipe.In.Did()
	outStr := tr.PLPipe.Out.Did()

	if inStr != tr.MsgPipe.In.Did() {
		inStr += "/" + tr.MsgPipe.In.Did()
	}

	inWallet := strconv.Itoa(tr.PLPipe.In.Wallet())
	if tr.PLPipe.In.Wallet() != tr.MsgPipe.In.Wallet() {
		inWallet += "/" + strconv.Itoa(tr.MsgPipe.In.Wallet())
	}

	if outStr != tr.MsgPipe.Out.Did() {
		outStr += "/" + tr.MsgPipe.Out.Did()
	}

	return fmt.Sprintf("In: %s(%s) Out: %s Ep: %s",
		inStr, inWallet, outStr, tr.Endp)
}

func (tr Transport) EndpAddr() string {
	return tr.Endp
}

func (tr Transport) PayloadPipe() sec.Pipe {
	return tr.PLPipe
}

func (tr Transport) MessagePipe() sec.Pipe {
	return tr.MsgPipe
}

func (tr Transport) Endpoint() string {
	return tr.Endp
}

func (tr Transport) SetMessageOut(d *ssi.DID) {
	tr.MsgPipe.Out = d
}

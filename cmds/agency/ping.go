package agency

import (
	"errors"
	"io"

	"github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/lainio/err2"
)

type PingCmd struct {
	BaseAddr string
}

func (c PingCmd) Validate() error {
	if c.BaseAddr == "" {
		return errors.New("server url cannot be empty")
	}
	return nil
}

func (c PingCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	defer err2.Return(&err)

	p := mesg.Payload{}

	endpointAdd := &endp.Addr{
		BasePath: c.BaseAddr,
		Service:  agency.APIPath,
		PlRcvr:   "ping",
	}

	pl := e2.Payload.Try(cmds.SendAndWaitPayload(&p, endpointAdd, 0))
	cmds.Fprintln(w, "ping ok.",
		"\nserver's host address:", pl.Message.Encrypted,
		"\nversion info:", pl.Message.Name)

	return nil, nil
}

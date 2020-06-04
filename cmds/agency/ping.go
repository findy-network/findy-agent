package agency

import (
	"errors"
	"io"

	"github.com/lainio/err2"
	"github.com/optechlab/findy-agent/agent/agency"
	"github.com/optechlab/findy-agent/agent/e2"
	"github.com/optechlab/findy-agent/agent/endp"
	"github.com/optechlab/findy-agent/agent/mesg"
	"github.com/optechlab/findy-agent/cmds"
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

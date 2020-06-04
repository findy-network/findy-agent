package connection

import (
	"io"

	"github.com/optechlab/findy-agent/agent/mesg"
	"github.com/optechlab/findy-agent/agent/pltype"
	"github.com/optechlab/findy-agent/cmds"
)

type TrustPingCmd struct {
	Cmd
}

func (c TrustPingCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	return c.Cmd.Exec(w, pltype.CATrustPing, &mesg.Msg{Name: c.Name})
}

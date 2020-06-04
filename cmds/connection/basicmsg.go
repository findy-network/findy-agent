package connection

import (
	"io"

	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/cmds"
)

type BasicMsgCmd struct {
	Cmd
	Message string
	Sender  string
}

func (c BasicMsgCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	return c.Cmd.Exec(w, pltype.CABasicMessage, &mesg.Msg{
		Name: c.Name,
		Info: c.Message,
		ID:   c.Sender,
	})
}

package sa

import (
	"errors"
	"io"

	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/sa"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-agent/cmds/agent"
	"github.com/lainio/err2"
)

type EAImplCmd struct {
	cmds.Cmd
	EAImplID string
}

func (c EAImplCmd) Validate() error {
	if err := c.Cmd.Validate(); err != nil {
		return err
	}
	if err := c.Cmd.ValidateWalletExistence(true); err != nil {
		return err
	}
	if !sa.Exists(c.EAImplID) {
		return errors.New("given EA implementation ID is not registered")
	}
	return nil
}

func (c EAImplCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	return c.Cmd.Exec(w, pltype.CAAttachEADefImpl, &mesg.Msg{
		ID: c.EAImplID,
	},
		func(_ cmds.Edge, im *mesg.Msg) (cmds.Result, error) {
			defer err2.Annotate("EA impl", &err)
			cmds.Fprintln(w, "EA implementation successfully set")
			return &agent.Result{}, nil
		})
}

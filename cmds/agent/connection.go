package agent

import (
	"io"

	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-agent/cmds/connection"
	didexchange "github.com/findy-network/findy-agent/std/didexchange/invitation"
)

type ConnectionCmd struct {
	cmds.Cmd
	didexchange.Invitation
}

func (c ConnectionCmd) Validate() error {
	if err := c.Cmd.Validate(); err != nil {
		return err
	}
	if err := c.Cmd.ValidateWalletExistence(true); err != nil {
		return err
	}
	if c.ID == "" || c.Label == "" || len(c.RecipientKeys) == 0 ||
		c.ServiceEndpoint == "" {
		return cmds.ErrInvalid
	}
	return nil
}

func (c ConnectionCmd) Exec(progress io.Writer) (r cmds.Result, err error) {
	// note! Read carefully, we create a new command from connection package
	// and route the cmd execution to that one
	return connection.Cmd{
		Cmd:  c.Cmd,
		Name: c.Label,
	}.Exec(progress, pltype.CAPairwiseCreate, &mesg.Msg{
		Info:       c.Label,
		Invitation: &c.Invitation,
	})
}

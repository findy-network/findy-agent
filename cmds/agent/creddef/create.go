package creddef

import (
	"errors"
	"io"

	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/cmds"
)

type CreateCmd struct {
	cmds.Cmd
	SchemaID string
	Tag      string
}

func (c CreateCmd) Validate() error {
	if err := c.Cmd.Validate(); err != nil {
		return err
	}
	if err := c.Cmd.ValidateWalletExistence(true); err != nil {
		return err
	}
	if len(c.SchemaID) == 0 {
		return errors.New("schema ID cannot be empty")
	}
	if len(c.Tag) == 0 {
		return errors.New("Tag cannot be empty")
	}
	return nil
}

type CreateResult struct {
	ID string
}

func (r CreateResult) JSON() ([]byte, error) {
	return nil, nil
}

func (c CreateCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	return c.Cmd.Exec(w, pltype.CACredDefCreate, &mesg.Msg{
		Schema: &ssi.Schema{ID: c.SchemaID},
		Info:   c.Tag,
	}, func(_ cmds.Edge, im *mesg.Msg) (cmds.Result, error) {
		cmds.Fprintln(w, im.ID)
		return &CreateResult{ID: im.ID}, nil
	})
}

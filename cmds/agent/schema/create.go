package schema

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/optechlab/findy-agent/agent/mesg"
	"github.com/optechlab/findy-agent/agent/pltype"
	"github.com/optechlab/findy-agent/agent/ssi"
	"github.com/optechlab/findy-agent/cmds"
)

type CreateCmd struct {
	cmds.Cmd
	*ssi.Schema
}

func (c CreateCmd) Validate() error {
	if err := c.Cmd.Validate(); err != nil {
		return err
	}
	if err := c.Cmd.ValidateWalletExistence(true); err != nil {
		return err
	}
	if len(c.Schema.Attrs) == 0 {
		return errors.New("attrs cannot be empty")
	}
	return nil
}

type CreateResult struct {
	*ssi.Schema
}

func (r CreateResult) JSON() ([]byte, error) {
	return json.Marshal(r.Schema)
}

func (c CreateCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	return c.Cmd.Exec(w, pltype.CASchemaCreate, &mesg.Msg{
		Schema: c.Schema,
	}, func(_ cmds.Edge, im *mesg.Msg) (cmds.Result, error) {
		cmds.Fprintln(w, "schema id:", im.ID)
		return &CreateResult{Schema: im.Schema}, nil
	})
}

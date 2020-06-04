package schema

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/cmds"
)

type GetCmd struct {
	cmds.Cmd
	ID string
}

func (c GetCmd) Validate() error {
	if err := c.Cmd.Validate(); err != nil {
		return err
	}
	if err := c.Cmd.ValidateWalletExistence(true); err != nil {
		return err
	}
	if len(c.ID) == 0 {
		return errors.New("schema ID cannot be empty")
	}
	return nil
}

type GetResult struct {
	Schema map[string]interface{}
}

func (r GetResult) JSON() ([]byte, error) {
	return json.Marshal(r.Schema)
}

func (c GetCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	return c.Cmd.Exec(w, pltype.CALedgerGetSchema, &mesg.Msg{
		ID: c.ID,
	}, func(_ cmds.Edge, im *mesg.Msg) (cmds.Result, error) {
		cmds.Fprintln(w, "schema id:", im.ID)
		return &GetResult{Schema: im.Msg}, nil
	})
}

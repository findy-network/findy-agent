package creddef

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/optechlab/findy-agent/agent/mesg"
	"github.com/optechlab/findy-agent/agent/pltype"
	"github.com/optechlab/findy-agent/cmds"
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
		return errors.New("cred def ID cannot be empty")
	}
	return nil
}

type GetResult struct {
	CredDef map[string]interface{}
}

func (r GetResult) JSON() ([]byte, error) {
	return json.Marshal(r.CredDef)
}

func (c GetCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	return c.Cmd.Exec(w, pltype.CALedgerGetCredDef, &mesg.Msg{
		ID: c.ID,
	}, func(_ cmds.Edge, im *mesg.Msg) (cmds.Result, error) {
		cmds.Fprintln(w, "cred def id:", im.ID)
		return &GetResult{CredDef: im.Msg}, nil
	})
}

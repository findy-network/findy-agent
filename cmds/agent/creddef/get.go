package creddef

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/lainio/err2"
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
	}, func(_ cmds.Edge, im *mesg.Msg) (r cmds.Result, err error) {
		defer err2.Return(&err)

		result := &GetResult{CredDef: im.Msg}
		jstr := string(err2.Bytes.Try(result.JSON()))
		cmds.Fprintln(w, jstr)

		return result, nil
	})
}

package agent

import (
	"errors"
	"io"

	"github.com/findy-network/findy-agent/agent/cloud"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/lainio/err2"
)

type ExportCmd struct {
	cmds.Cmd
	Filename  string
	ExportKey string
}

func (c ExportCmd) Validate() error {
	if err := c.Cmd.Validate(); err != nil {
		return err
	}
	if err := c.Cmd.ValidateWalletExistence(true); err != nil {
		return err
	}
	if c.Filename == "" {
		return errors.New("export path cannot be empty")
	}
	if err := cmds.ValidateKey(c.ExportKey); err != nil {
		return err
	}
	return nil
}

func (c ExportCmd) Exec(w io.Writer) (r Result, err error) {
	defer err2.Annotate("export wallet cmd", &err)

	agent := cloud.NewEA()
	agent.OpenWallet(*ssi.NewRawWalletCfg(c.WalletName, c.WalletKey))
	defer agent.CloseWallet()

	agent.ExportWallet(c.ExportKey, c.Filename)
	err2.Check(agent.Export.Result().Err())

	cmds.Fprintln(w, "wallet exported:", c.Filename)
	return Result{}, nil
}

package tools

import (
	"errors"
	"io"

	"github.com/findy-network/findy-agent/agent/cloud"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

type ExportCmd struct {
	cmds.Cmd

	WalletKeyLegacy bool

	Filename  string
	ExportKey string
}

func (c ExportCmd) Validate() error {
	if !c.WalletKeyLegacy {
		if err := c.Cmd.Validate(); err != nil {
			return err
		}
		if err := c.Cmd.ValidateWalletExistence(true); err != nil {
			return err
		}
	} else {
		exists := ssi.NewWalletCfg(c.WalletName, c.WalletKey).Exists(false)
		if !exists {
			return errors.New("legacy wallet not exist")
		}
	}
	if c.Filename == "" {
		return errors.New("export path cannot be empty")
	}
	return cmds.ValidateKey(c.ExportKey, "export")
}

func (c ExportCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	defer err2.Handle(&err, "export wallet cmd")

	agent := cloud.NewEA()
	wallet := *ssi.NewRawWalletCfg(c.WalletName, c.WalletKey)
	if c.WalletKeyLegacy {
		wallet = *ssi.NewWalletCfg(c.WalletName, c.WalletKey)
	}
	agent.OpenWallet(wallet)
	defer agent.CloseWallet()

	agent.ExportWallet(c.ExportKey, c.Filename)
	try.To(agent.Export.Result().Err())

	cmds.Fprintln(w, "wallet exported:", c.Filename)
	return r, nil
}

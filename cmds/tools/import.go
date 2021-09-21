package tools

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-wrapper-go/wallet"
	"github.com/lainio/err2"
)

type ImportCmd struct {
	cmds.Cmd
	Filename string
	Key      string
}

func (c ImportCmd) Validate() error {
	if err := c.Cmd.Validate(); err != nil {
		return err
	}
	if err := c.Cmd.ValidateWalletExistence(false); err != nil {
		return err
	}
	if c.Filename == "" {
		return errors.New("export path cannot be empty")
	}
	if err := cmds.ValidateKey(c.Key, "import"); err != nil {
		return err
	}
	_, err := os.Stat(c.Filename)
	if os.IsNotExist(err) {
		return fmt.Errorf("file: %v not exist", c.Filename)
	}

	return nil
}

func (c ImportCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	defer err2.Annotate("import wallet cmd", &err)

	walletCfg := wallet.Config{
		ID: c.WalletName,
	}
	walletCreds := wallet.Credentials{
		Key:                 c.WalletKey,
		KeyDerivationMethod: "RAW",
	}
	importCreds := wallet.Credentials{
		Path: c.Filename,
		Key:  c.Key,
	}

	res := <-wallet.Import(walletCfg, walletCreds, importCreds)
	err2.Check(res.Err())

	cmds.Fprintf(w, "wallet %s imported from file %s\n", c.WalletName,
		c.Filename)
	return r, nil
}

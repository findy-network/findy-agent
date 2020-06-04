package key

import (
	"io"

	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-wrapper-go/wallet"
	"github.com/lainio/err2"
)

type CreateCmd struct {
	Seed string
}

func (c *CreateCmd) Validate() error {
	if err := cmds.ValidateSeed(c.Seed); err != nil {
		return err
	}
	return nil
}

func (c *CreateCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	err2.Return(&err)

	result := <-wallet.GenerateKey(c.Seed)
	err2.Check(result.Err())
	walletKey := result.Str1()
	cmds.Fprintln(w, walletKey)

	return r, nil
}

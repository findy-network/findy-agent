package key

import (
	"io"

	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-wrapper-go/wallet"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
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

type CreateResult struct {
	Key string
}

func (r CreateResult) JSON() ([]byte, error) {
	return nil, nil
}

func (c *CreateCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	defer err2.Return(&err)

	result := <-wallet.GenerateKey(c.Seed)
	try.To(result.Err())
	walletKey := result.Str1()
	cmds.Fprintln(w, walletKey)

	return &CreateResult{Key: walletKey}, nil
}

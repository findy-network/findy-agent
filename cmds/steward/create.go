package steward

import (
	"errors"
	"io"

	"github.com/findy-network/findy-agent/agent/async"
	"github.com/findy-network/findy-agent/agent/cloud"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-wrapper-go/did"
	"github.com/lainio/err2"
)

type CreateCmd struct {
	cmds.Cmd
	PoolName    string
	StewardSeed string
}

func (c *CreateCmd) Validate() error {
	if err := c.Cmd.Validate(); err != nil {
		return err
	}
	if err := c.Cmd.ValidateWalletExistence(false); err != nil {
		return err
	}

	if c.PoolName == "" {
		return errors.New("pool name cannot be empty")
	}

	return cmds.ValidateSeed(c.StewardSeed)
}

func (c *CreateCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	defer err2.Handle(&err)

	stewardAgent := new(cloud.Agent)
	stewardAgent.OpenPool(c.PoolName)
	_ = stewardAgent.Pool() // read from future

	agentWallet := ssi.NewRawWalletCfg(c.WalletName, c.WalletKey)
	agentWallet.Create()
	walletFuture := agentWallet.Open()

	walletFuture.Int()

	var seed string
	if c.StewardSeed != "" {
		seed = c.StewardSeed
	}
	f := new(async.Future)
	f.SetChan(did.CreateAndStore(walletFuture.Int(), did.Did{Seed: seed}))

	cmds.Fprintln(w,
		"steward DID:", f.Str1(),
		"\nsteward VerKey:", f.Str2())

	return r, nil
}

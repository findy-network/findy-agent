package agent

import (
	"io"

	"github.com/findy-network/findy-agent/agent/cloud"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/trans"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/lainio/err2"
)

type NotifyChan chan mesg.Payload

type ListenCmd struct {
	cmds.Cmd
	NotifyChan
}

func (c ListenCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	return c.exec(w)
}

func (c ListenCmd) exec(p io.Writer) (r cmds.Result, err error) {
	defer err2.Return(&err)

	wcfg := ssi.NewRawWalletCfg(c.WalletName, c.WalletKey)
	edge := cmds.Edge{
		Cmd:   c.Cmd,
		Agent: cloud.NewTransportReadyEA(wcfg),
	}
	defer edge.Agent.CloseWallet()

	tr := edge.Agent.Trans().(trans.Transport)
	err2.Check(tr.WsListenLoop("ca-apiws",
		func(im *mesg.Payload) (while bool, err error) {
			defer err2.Return(&err)
			c.NotifyChan <- *im
			return true, nil
		}))
	return &Result{}, nil
}

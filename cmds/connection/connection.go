package connection

import (
	"errors"
	"io"
	"time"

	"github.com/findy-network/findy-agent/agent/cloud"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/trans"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/lainio/err2"
)

// timeout to wait a task before we stop. When real ledger is in use, this must
// be quite high.
const timeout = 8000 * time.Millisecond

type Cmd struct {
	cmds.Cmd
	Name string
}

func (c Cmd) Validate() error {
	if err := c.Cmd.Validate(); err != nil {
		return err
	}
	if err := c.Cmd.ValidateWalletExistence(true); err != nil {
		return err
	}
	if c.Name == "" {
		return errors.New("connection name cannot be empty")
	}
	return nil
}

type Result struct {
	TaskID string
	Ready  bool
}

func (r Result) JSON() ([]byte, error) {
	return nil, nil
}

func (c Cmd) Exec(p io.Writer, t string, m *mesg.Msg) (r cmds.Result, err error) {
	defer err2.Return(&err)

	readyTask := make(chan string)
	started := make(chan struct{})

	wcfg := ssi.NewRawWalletCfg(c.WalletName, c.WalletKey)
	edge := cmds.Edge{
		Cmd:   c.Cmd,
		Agent: cloud.NewTransportReadyEA(wcfg),
	}
	defer edge.Agent.CloseWallet()

	tr := edge.Agent.Trans().(trans.Transport)
	go func() {
		started <- struct{}{}
		err2.Check(tr.WsListenLoop("ca-apiws",
			func(im *mesg.Payload) (while bool, err error) {
				defer err2.Return(&err)

				switch im.Type {
				case pltype.CANotifyStatus:
					readyTask <- im.Message.Nonce
				}
				return true, nil
			}))
	}()
	<-started
	im := e2.Msg.Try(edge.MsgToCA(p, t, m))
	select {
	case tID := <-readyTask:
		if im.ID == tID {
			return &Result{TaskID: tID, Ready: true}, nil
		}
	case <-time.After(timeout):
	}
	return &Result{TaskID: im.ID}, nil
}

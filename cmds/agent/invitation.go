package agent

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/lainio/err2"
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/cmds"
	didexchange "github.com/findy-network/findy-agent/std/didexchange/invitation"
)

type InvitationCmd struct {
	cmds.Cmd
	ID   string
	Name string
}

func (c InvitationCmd) Validate() error {
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

type InvitationResult struct {
	didexchange.Invitation
}

func (r InvitationResult) JSON() ([]byte, error) {
	return json.Marshal(&r.Invitation)
}

func (c InvitationCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	return c.Cmd.Exec(w, pltype.CAPingOwnCA, &mesg.Msg{},
		func(e cmds.Edge, im *mesg.Msg) (cmds.Result, error) {
			defer err2.Annotate("invitation", &err)

			ep := endp.NewClientAddr(im.Endpoint)
			ep.RcvrDID = e.Agent.Trans().MessagePipe().In.Did()

			id := c.ID
			if id == "" {
				id = utils.UUID()
			}
			invitation := didexchange.Invitation{
				ID:              id,
				Type:            pltype.AriesConnectionInvitation,
				ServiceEndpoint: ep.Address(),
				RecipientKeys:   []string{e.Agent.Tr.PayloadPipe().Out.VerKey()},
				Label:           c.Name,
			}

			ir := &InvitationResult{Invitation: invitation}
			jsonBytes := err2.Bytes.Try(ir.JSON())
			cmds.Fprintln(w, string(jsonBytes))

			return ir, nil
		})
}

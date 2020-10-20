package sa

import (
	"errors"
	"io"
	"net/url"

	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/sa"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-agent/cmds/agent"
	"github.com/lainio/err2"
)

type EAImplCmd struct {
	cmds.Cmd
	EAImplID     string
	EAServiceURL string
	EAServiceKey string
}

func (c EAImplCmd) Validate() error {
	if err := c.Cmd.Validate(); err != nil {
		return err
	}
	if err := c.Cmd.ValidateWalletExistence(true); err != nil {
		return err
	}
	if c.EAServiceURL == "" && c.EAImplID == "" {
		return errors.New("EA endpoint (url+key) OR implementation ID missing")
	}
	if c.EAServiceURL != "" {
		if err := cmds.ValidateKey(c.EAServiceKey, "endpoint"); err != nil {
			return err
		}
		if _, err := url.ParseRequestURI(c.EAServiceURL); err != nil {
			return err
		}
	} else if !sa.Exists(c.EAImplID) {
		return errors.New("given EA implementation ID is not registered")
	}
	return nil
}

func (c EAImplCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	if c.EAServiceURL != "" {
		return c.Cmd.Exec(w, pltype.CAAttachAPIEndp, &mesg.Msg{
			RcvrEndp: c.EAServiceURL,
			RcvrKey:  c.EAServiceKey,
			Ready:    true, // we want tasks notifications to this endpoint
		},
			func(_ cmds.Edge, im *mesg.Msg) (cmds.Result, error) {
				defer err2.Annotate("EA endpoint", &err)
				cmds.Fprintln(w, "EA endpoint successfully set")
				return &agent.Result{}, nil
			})
	}
	return c.Cmd.Exec(w, pltype.CAAttachEADefImpl, &mesg.Msg{
		ID: c.EAImplID,
	},
		func(_ cmds.Edge, im *mesg.Msg) (cmds.Result, error) {
			defer err2.Annotate("EA impl", &err)
			cmds.Fprintln(w, "EA implementation successfully set")
			return &agent.Result{}, nil
		})
}

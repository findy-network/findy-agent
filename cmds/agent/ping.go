package agent

import (
	"io"

	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-grpc/jwt"
	"github.com/lainio/err2"
)

type PingCmd struct {
	cmds.Cmd
	PingSA  bool
	DIDOnly bool
	JWT     bool
}

func (c PingCmd) Validate() error {
	if err := c.Cmd.Validate(); err != nil {
		return err
	}
	if err := c.Cmd.ValidateWalletExistence(true); err != nil {
		return err
	}
	return nil
}

func (c PingCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	if c.PingSA {
		return c.Cmd.Exec(w, pltype.CAPingAPIEndp, &mesg.Msg{},
			func(_ cmds.Edge, im *mesg.Msg) (cmds.Result, error) {
				defer err2.Annotate("ping ca", &err)
				if im.Ready {
					cmds.Fprintln(w, "SA ping ok")
				} else {
					cmds.Fprintln(w, "SA ping error:", im.Info)
				}
				return &Result{}, nil
			})
	}
	return c.Cmd.Exec(w, pltype.CAPingOwnCA, &mesg.Msg{},
		func(_ cmds.Edge, im *mesg.Msg) (cmds.Result, error) {
			defer err2.Annotate("ping sa", &err)
			if c.DIDOnly {
				did := im.Did
				if did == "" {
					ea := endp.NewClientAddr(im.Endpoint)
					did = ea.ReceiverDID()
				}
				if c.JWT {
					cmds.Fprint(w, jwt.BuildJWT(did))
				} else {
					cmds.Fprint(w, did)
				}
			} else {
				cmds.Fprintln(w, "Endpoint from the server:")
				cmds.Fprintln(w, im.Endpoint)
				cmds.Fprintln(w, "Verkey from the server:")
				cmds.Fprintln(w, im.EndpVerKey)
			}
			return &Result{}, nil
		})
}

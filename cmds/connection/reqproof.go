package connection

import (
	"encoding/json"
	"io"

	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/cmds"
)

type ReqProofCmd struct {
	Cmd
	Attributes string
}

func (c ReqProofCmd) Validate() error {
	if err := c.Cmd.Validate(); err != nil {
		return err
	}

	if _, err := parseProofAttrs(c.Attributes); err != nil {
		return err
	}
	return nil
}

func parseProofAttrs(a string) (proofAttrs []didcomm.ProofAttribute, err error) {
	if err := json.Unmarshal([]byte(a), &proofAttrs); err != nil {
		return nil, err
	}
	return proofAttrs, nil
}

func (c ReqProofCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	proofAttrs, err := parseProofAttrs(c.Attributes)
	if err != nil {
		return nil, err
	}

	return c.Cmd.Exec(w, pltype.CAProofRequest, &mesg.Msg{
		Name:       c.Name,
		ProofAttrs: &proofAttrs,
	})
}

package connection

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/cmds"
)

type IssueCmd struct {
	Cmd
	CredDefID  string
	Attributes string
}

func (c IssueCmd) Validate() error {
	if err := c.Cmd.Validate(); err != nil {
		return err
	}
	if c.CredDefID == "" {
		return errors.New("cred def id cannot be empty")
	}

	if _, err := parseAttrs(c.Attributes); err != nil {
		return err
	}
	return nil
}

func parseAttrs(a string) (credAttrs []didcomm.CredentialAttribute, err error) {
	if err := json.Unmarshal([]byte(a), &credAttrs); err != nil {
		return nil, err
	}
	return credAttrs, nil
}

func (c IssueCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	credAttrs, err := parseAttrs(c.Attributes)
	if err != nil {
		return nil, err
	}

	return c.Cmd.Exec(w, pltype.CACredOffer, &mesg.Msg{
		Name:            c.Name,
		CredDefID:       &c.CredDefID,
		CredentialAttrs: &credAttrs,
	})
}

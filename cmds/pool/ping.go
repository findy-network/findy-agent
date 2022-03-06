package pool

import (
	"io"

	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/lainio/err2"
)

type PingCmd struct {
	Name string
}

func (c *PingCmd) Validate() error {
	if c.Name == "" {
		return cmds.ErrInvalid
	}
	return nil
}

func (c *PingCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	defer err2.Return(&err)

	cmds.Fprintln(w, "starting to open cnx to:", c.Name)
	h := ssi.OpenPool(c.Name).Int()
	cmds.Fprintln(w, "pool handle:", h)
	ssi.ClosePool()
	cmds.Fprintln(w, "pool closed")

	return r, nil
}

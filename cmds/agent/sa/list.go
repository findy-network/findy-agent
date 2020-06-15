package sa

import (
	"io"

	"github.com/findy-network/findy-agent/agent/sa"
	"github.com/findy-network/findy-agent/cmds"
)

type ListCmd struct{}

func (c ListCmd) Validate() error {
	return nil
}

type ListResult struct {
	Implementations []string
}

func (r ListResult) JSON() ([]byte, error) {
	return nil, nil
}

func (c ListCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	list := sa.List()
	for _, s := range list {
		cmds.Fprintln(w, s)
	}
	return &ListResult{Implementations: list}, nil
}

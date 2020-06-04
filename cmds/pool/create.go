package pool

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/findy-network/findy-agent/cmds"
	findypool "github.com/findy-network/findy-wrapper-go/pool"
	"github.com/lainio/err2"
)

type CreateCmd struct {
	Name string
	Txn  string
}

func (c *CreateCmd) Validate() error {
	if c.Name == "" {
		return errors.New("pool name cannot be empty")
	}
	if c.Name == "FINDY_MEM_LEDGER" || c.Name == "FINDY_ECHO_LEDGER" {
		return fmt.Errorf("%s is not a valid ledger name", c.Name)
	}
	if c.Txn == "" {
		return errors.New("pool genesis file is required")
	}
	_, err := os.Stat(c.Txn)
	if os.IsNotExist(err) {
		return errors.New("pool genesis does not exist")
	}
	return nil
}

func (c *CreateCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	err2.Return(&err)

	pCfd := <-findypool.CreateConfig(c.Name, findypool.Config{GenesisTxn: c.Txn})
	if pCfd.Err() != nil {
		cmds.Fprintln(w, "pool creation ERROR, EXITING")
		cmds.Fprintln(w, pCfd.Err())
		panic(pCfd.Err())
	}
	cmds.Fprintln(w, "Pool created successfully by name:", c.Name)

	return r, nil
}

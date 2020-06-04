package e2

import (
	"github.com/lainio/err2"
	"github.com/findy-network/findy-agent/agent/didcomm"
)

type _M struct{}

// M is a helper variable to generated
// 'type wrappers' to make Try function as fast as Check.
var M _M

// Try is a helper method to call func() (didcomm.Msg, error) functions
// with it and be as fast as Check(err).
func (o _M) Try(v didcomm.Msg, err error) didcomm.Msg {
	err2.Check(err)
	return v
}

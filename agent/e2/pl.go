package e2

import (
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/lainio/err2"
)

type _PL struct{}

// PL is a helper variable to generated
// 'type wrappers' to make Try function as fast as Check.
var PL _PL

// Try is a helper method to call func() (didcomm.Payload, error) functions
// with it and be as fast as Check(err).
func (o _PL) Try(v didcomm.Payload, err error) didcomm.Payload {
	err2.Check(err)
	return v
}

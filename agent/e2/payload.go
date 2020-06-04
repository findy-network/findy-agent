package e2

import (
	"github.com/lainio/err2"
	"github.com/findy-network/findy-agent/agent/mesg"
)

type _Payload struct{}

// Payload is a helper variable to generated
// 'type wrappers' to make Try function as fast as Check.
var Payload _Payload

// Try is a helper method to call func() (*mesg.Payload, error) functions
// with it and be as fast as Check(err).
func (o _Payload) Try(v *mesg.Payload, err error) *mesg.Payload {
	err2.Check(err)
	return v
}

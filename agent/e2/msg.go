package e2

import (
	"github.com/lainio/err2"
	"github.com/optechlab/findy-agent/agent/mesg"
)

type _Msg struct{}

// Msg is a helper variable to generated
// 'type wrappers' to make Try function as fast as Check.
var Msg _Msg

// Try is a helper method to call func() (mesg.Msg, error) functions
// with it and be as fast as Check(err).
func (o _Msg) Try(v mesg.Msg, err error) mesg.Msg {
	err2.Check(err)
	return v
}

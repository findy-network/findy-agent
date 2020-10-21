package e2

import (
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/lainio/err2"
)

type _Rcvr struct{}

// Rcvr is a helper variable to generated
// 'type wrappers' to make Try function as fast as Check.
var Rcvr _Rcvr

// Try is a helper method to call func() (comm.Receiver, error) functions
// with it and be as fast as Check(err).
func (o _Rcvr) Try(v comm.Receiver, err error) comm.Receiver {
	err2.Check(err)
	return v
}

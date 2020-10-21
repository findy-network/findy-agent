package e2

import (
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/lainio/err2"
)

type _StrRcvr struct{}

// Rcvr is a helper variable to generated
// 'type wrappers' to make Try function as fast as Check.
var StrRcvr _StrRcvr

// Try is a helper method to call func() (str, comm.Receiver, error) functions
// with it and be as fast as Check(err).
func (o _StrRcvr) Try(s string, v comm.Receiver, err error) (string, comm.Receiver) {
	err2.Check(err)
	return s, v
}

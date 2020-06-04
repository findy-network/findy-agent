package e2

import (
	"github.com/lainio/err2"
	"github.com/optechlab/findy-agent/agent/sec"
)

type _Pipe struct{}

// Pipe is a helper variable to generated
// 'type wrappers' to make Try function as fast as Check.
var Pipe _Pipe

// Try is a helper method to call func() (sec.Pipe, error) functions
// with it and be as fast as Check(err).
func (o _Pipe) Try(v sec.Pipe, err error) sec.Pipe {
	err2.Check(err)
	return v
}

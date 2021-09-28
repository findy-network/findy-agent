package e2

import (
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/lainio/err2"
)

type _Task struct{}

// Task is a helper variable to generated
// 'type wrappers' to make Try function as fast as Check.
var Task _Task

// Try is a helper method to call func() (*comm.Task, error) functions
// with it and be as fast as Check(err).
func (o _Task) Try(v comm.Task, err error) comm.Task {
	err2.Check(err)
	return v
}

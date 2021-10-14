package e2

import (
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/lainio/err2"
)

type _Public struct{}

// Public is a helper variable to generated
// 'type wrappers' to make Try function as fast as Check.
var Public _Public

// Try is a helper method to call func() (service.Addr, error) functions
// with it and be as fast as Check(err).
func (o _Public) Try(v service.Addr, err error) service.Addr {
	err2.Check(err)
	return v
}

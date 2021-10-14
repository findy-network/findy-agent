package e2

import (
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/lainio/err2"
)

type _Addr struct{}

// Addr is a helper variable to generated
// 'type wrappers' to make Try function as fast as Check.
var Addr _Addr

// Try is a helper method to call func() (endp.Addr, error) functions
// with it and be as fast as Check(err).
func (o _Addr) Try(v *endp.Addr, err error) *endp.Addr {
	err2.Check(err)
	return v
}

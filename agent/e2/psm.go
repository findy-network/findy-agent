package e2

import (
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/lainio/err2"
)

type _PSM struct{}

// PSM is a helper variable to generated
// 'type wrappers' to make Try function as fast as Check.
var PSM _PSM

// Try is a helper method to call func() (*psm.PSM, error) functions
// with it and be as fast as Check(err).
func (o _PSM) Try(v *psm.PSM, err error) *psm.PSM {
	err2.Check(err)
	return v
}

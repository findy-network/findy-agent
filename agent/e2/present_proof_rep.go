package e2

import (
	"github.com/findy-network/findy-agent/protocol/presentproof/rep"
	"github.com/lainio/err2"
)

type _PresentProofRep struct{}

// PresentProofRep is a helper variable to generated
// 'type wrappers' to make Try function as fast as Check.
var PresentProofRep _PresentProofRep

// Try is a helper method to call func() (*psm.PresentProofRep, error) functions
// with it and be as fast as Check(err).
func (o _PresentProofRep) Try(v *rep.PresentProofRep, err error) *rep.PresentProofRep {
	err2.Check(err)
	return v
}

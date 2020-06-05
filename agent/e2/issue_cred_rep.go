package e2

import (
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/lainio/err2"
)

type _IssueCredRep struct{}

// IssueCredRep is a helper variable to generated
// 'type wrappers' to make Try function as fast as Check.
var IssueCredRep _IssueCredRep

// Try is a helper method to call func() (*psm.IssueCredRep, error) functions
// with it and be as fast as Check(err).
func (o _IssueCredRep) Try(v *psm.IssueCredRep, err error) *psm.IssueCredRep {
	err2.Check(err)
	return v
}

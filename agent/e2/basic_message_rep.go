package e2

type _BasicMessageRep struct{}

// BasicMessageRep is a helper variable to generated
// 'type wrappers' to make Try function as fast as Check.
var BasicMessageRep _BasicMessageRep

// Try is a helper method to call func() (*psm.BasicMessageRep, error) functions
// with it and be as fast as Check(err).
/*func (o _BasicMessageRep) Try(v *psm.BasicMessageRep, err error) *psm.BasicMessageRep {
	err2.Check(err)
	return v
}*/

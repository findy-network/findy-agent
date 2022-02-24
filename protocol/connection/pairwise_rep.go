package connection

import (
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
)

const bucketType = psm.BucketPairwise

type pairwiseRep struct {
	psm.StateKey
	Name       string // In our implementation this is connection id!
	TheirLabel string
	Caller     didRep
	Callee     didRep
}

func init() {
	psm.Creator.Add(bucketType, NewPairwiseRep)
}

func NewPairwiseRep(d []byte) psm.Rep {
	p := &pairwiseRep{}
	dto.FromGOB(d, p)
	return p
}

func (p *pairwiseRep) Key() psm.StateKey {
	return p.StateKey
}

func (p *pairwiseRep) Data() []byte {
	return dto.ToGOB(p)
}

func (p *pairwiseRep) Type() byte {
	return bucketType
}

func getPairwiseRep(key psm.StateKey) (rep *pairwiseRep, err error) {
	err2.Return(&err)

	var res psm.Rep
	res, err = psm.GetRep(bucketType, key)
	err2.Check(err)

	var ok bool
	rep, ok = res.(*pairwiseRep)

	assert.D.True(ok, "pairwise type mismatch")

	return rep, nil
}

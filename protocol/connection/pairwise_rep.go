package connection

import (
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
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
	defer err2.Returnf(&err, "pairwise not found %s", key)

	res := try.To1(psm.GetRep(bucketType, key))

	rep, ok := res.(*pairwiseRep)
	assert.That(ok, "pairwise type mismatch")

	return rep, nil
}

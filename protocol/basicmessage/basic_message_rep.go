package basicmessage

import (
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

const bucketType = psm.BucketBasicMessage

type basicMessageRep struct {
	psm.StateKey
	PwName        string
	Message       string
	SendTimestamp int64
	Timestamp     int64
	SentByMe      bool
	Delivered     bool
}

func init() {
	psm.Creator.Add(bucketType, NewBasicMessageRep)
}

func NewBasicMessageRep(d []byte) psm.Rep {
	p := &basicMessageRep{}
	dto.FromGOB(d, p)
	return p
}

func (p *basicMessageRep) Key() psm.StateKey {
	return p.StateKey
}

func (p *basicMessageRep) Data() []byte {
	return dto.ToGOB(p)
}

func (p *basicMessageRep) Type() byte {
	return bucketType
}

func getBasicMessageRep(workerDID, taskID string) (rep *basicMessageRep, err error) {
	defer err2.Handle(&err)

	res := try.To1(psm.GetRep(bucketType, psm.StateKey{
		DID:   workerDID,
		Nonce: taskID,
	}))

	bmrep, ok := res.(*basicMessageRep)
	assert.That(ok, "basic message type mismatch")

	return bmrep, nil
}

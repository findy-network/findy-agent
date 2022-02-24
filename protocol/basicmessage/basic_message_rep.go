package basicmessage

import (
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
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
	err2.Return(&err)

	var res psm.Rep
	res, err = psm.GetRep(bucketType, psm.StateKey{
		DID:   workerDID,
		Nonce: taskID,
	})
	err2.Check(err)

	var ok bool
	rep, ok = res.(*basicMessageRep)

	assert.D.True(ok, "basic message type mismatch")

	return rep, nil
}

package psm

import (
	"github.com/optechlab/findy-go/dto"
)

type BasicMessageRep struct {
	Key           StateKey
	PwName        string
	Message       string
	SendTimestamp int64
	Timestamp     int64
	SentByMe      bool
	Delivered     bool
}

func NewBasicMessageRep(d []byte) *BasicMessageRep {
	p := &BasicMessageRep{}
	dto.FromGOB(d, p)
	return p
}

func (p *BasicMessageRep) Data() []byte {
	return dto.ToGOB(p)
}

func (p *BasicMessageRep) KData() []byte {
	return p.Key.Data()
}

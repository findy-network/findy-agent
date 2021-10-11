package basicmessage

import (
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-wrapper-go/dto"
)

type basicMessageRep struct {
	psm.BaseRep
	PwName        string
	Message       string
	SendTimestamp int64
	Timestamp     int64
	SentByMe      bool
	Delivered     bool
}

func init() {
	
}


func NewBasicMessageRep(d []byte) *basicMessageRep {
	p := &basicMessageRep{}
	dto.FromGOB(d, p)
	return p
}

func (p *basicMessageRep) Type() string {
	return "BasicMessage"
}

package sa

import "github.com/findy-network/findy-agent/agent/didcomm"

var implReg = make(map[string]Handler)

func Add(implID string, f Handler) {
	implReg[implID] = f
}

func Get(implID string) Handler {
	return implReg[implID]
}

type Handler func(plType string, im didcomm.Msg) (om didcomm.Msg, err error)

package sa

import (
	"github.com/findy-network/findy-agent/agent/didcomm"
)

var implReg = make(map[string]Handler)

func Add(implID string, f Handler) {
	implReg[implID] = f
}

func Get(implID string) Handler {
	return implReg[implID]
}

func Exists(implID string) bool {
	_, found := implReg[implID]
	return found
}

func List() []string {
	l := make([]string, len(implReg))
	var i int
	for id := range implReg {
		l[i] = id
		i++
	}
	return l
}

type Handler func(WDID string, plType string, im didcomm.Msg) (om didcomm.Msg, err error)

package comm

import (
	"fmt"
	"sync"

	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/golang/glog"
)

type caAPIProcessor struct {
	handlers map[string]PlHandlerFunc
}

// Process delivers the protocol messages inside the Packet to correct protocol.
func (p *caAPIProcessor) Process(packet Packet) (opl didcomm.Payload) {
	glog.V(1).Info("API type " + packet.Payload.Type())

	handler, ok := p.handlers[packet.Payload.Type()]
	if !ok {
		glog.Info(string(packet.Payload.JSON()))
		glog.Error("!!!! No handler in processor !!!")
		return mesg.PayloadCreator.NewError(packet.Payload, fmt.Errorf("no handler"))
	}
	return handler(packet)
}

func (p *caAPIProcessor) Add(h map[string]PlHandlerFunc) {
	if p.handlers == nil {
		p.handlers = h
		return
	}

	for k, v := range h {
		p.handlers[k] = v
	}
	glog.V(1).Info("handler count: ", len(p.handlers))
}

// PlHandlerFunc is func type for protocol message handlers. We add them to
// protocol processors with the associated message type. See Payload.Type
type PlHandlerFunc func(packet Packet) (opl didcomm.Payload)

var caProc *caAPIProcessor
var caProcOnce sync.Once

// CloudAgentAPI returns CA API processor.
func CloudAgentAPI() *caAPIProcessor {
	caProcOnce.Do(func() {
		caProc = &caAPIProcessor{}
	})
	return caProc
}

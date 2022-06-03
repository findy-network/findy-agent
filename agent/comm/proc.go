package comm

import (
	"github.com/findy-network/findy-agent/agent/didcomm"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/golang/glog"
)

// processor is a controller of all the protocol handlers. It keeps
// track of all the protocols and delivers the message accordingly. The
// processor takes the message and finds the correct protocol which finds the
// correct handler function.
//
// The protocol message processing structure has 2 levels. First level has
// protocols and second level has the actual message handlers. Protocols are run
// to achieve certain protocol outcome. During the protocol both parties send
// and receive multiple messages. Protocols have state so to speak, but we don't
// handle state here. It's better that protocol handling functions handle the
// state by their selves.
type processor struct {
	// protHandlers is map to all protocols and their handlers. The key is the
	// protocol name in the Payload.Type
	protHandlers map[string]ProtHandler
}

// Process delivers the protocol messages inside the packet to correct protocol.
func (p *processor) Process(packet Packet) (err error) {
	handler, ok := p.protHandlers[packet.Payload.Protocol()]
	if !ok {
		glog.Errorf("No handler in processor for Type: %s\nPL:\n%s",
			packet.Payload.Type(),
			string(packet.Payload.JSON()))
		panic("No handler in processor")
	}
	return handler.Process(packet)
}

func (p *processor) Add(t string, proc ProtHandler) {
	if p.protHandlers == nil {
		p.protHandlers = make(map[string]ProtHandler)
	}
	p.protHandlers[t] = proc
}

// HandlerFunc is func type for protocol message handlers. We add them to
// protocol processors with the associated message type. See Payload.Type
type HandlerFunc func(packet Packet) (err error)

// ProtHandler is an interface for whole protocol. Where HandlerFunc is handler
// for protocol message, the protocol handler is whole protocol group, all of
// the same message family.
type ProtHandler interface {
	Process(packet Packet) (err error)
}

// ProtProc is a protocol processor. It is struct for protocol handlers.
// Instances of it are the actual protocol handlers. Just declare var and the
// needed msg handlers (HandlerFunc) and register it to the processor.
type ProtProc struct {
	Creator
	Starter
	Handlers map[string]HandlerFunc
	Continuator
	FillStatus
}

type Creator func(header *TaskHeader, protocol *pb.Protocol) (Task, error)

type Starter func(ca Receiver, t Task)

type Continuator func(ca Receiver, im didcomm.Msg)

type FillStatus func(workerDID string, taskID string, ps *pb.ProtocolStatus) *pb.ProtocolStatus

// Process delivers the protocol message inside the packet to correct protocol
// function.
func (p ProtProc) Process(packet Packet) (err error) {
	glog.V(1).Info("PROTOCOL type " + packet.Payload.Type())

	handler, ok := p.Handlers[packet.Payload.ProtocolMsg()]
	if !ok {
		glog.Info(string(packet.Payload.JSON()))
		s := "!!!! No handler in processor !!!"
		glog.Error(s)
		panic(s)
	}
	return handler(packet)
}

var Proc = &processor{}

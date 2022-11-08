package didexchange

import (
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/core"
	"github.com/hyperledger/aries-framework-go/pkg/crypto"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
)

// PwMsg is an interface for pairwise protocol messages.
// It abstracts the protocol message implementation and provides
// needed data accessors for connection protocol engine.
type PwMsg interface {
	didcomm.MessageHdr

	Endpoint() service.Addr
	Did() string
	VerKey() string
	Label() string
	DIDDocument() core.DIDDoc
	RoutingKeys() []string

	Verify(c crypto.Crypto, keyManager kms.KeyManager) error

	PayloadToSend(ourLabel string, ourDID core.DID) (didcomm.Payload, psm.SubState, error)
	PayloadToWait() (didcomm.Payload, psm.SubState)
}

type UnsupportedPwMsgBase struct{}

func (m *UnsupportedPwMsgBase) Endpoint() service.Addr {
	panic("unsupported")
}

func (m *UnsupportedPwMsgBase) Did() string {
	panic("unsupported")
}

func (m *UnsupportedPwMsgBase) VerKey() string {
	panic("unsupported")
}

func (m *UnsupportedPwMsgBase) Label() string {
	panic("unsupported")
}

func (m *UnsupportedPwMsgBase) DIDDocument() core.DIDDoc {
	panic("unsupported")
}

func (m *UnsupportedPwMsgBase) RoutingKeys() []string {
	panic("unsupported")
}

func (m *UnsupportedPwMsgBase) Verify(c crypto.Crypto, keyManager kms.KeyManager) error {
	panic("unsupported")
}

func (m *UnsupportedPwMsgBase) PayloadToSend(ourLabel string, ourDID core.DID) (didcomm.Payload, psm.SubState, error) {
	panic("unsupported")
}

func (m *UnsupportedPwMsgBase) PayloadToWait() (didcomm.Payload, psm.SubState) {
	panic("unsupported")
}

func (m *UnsupportedPwMsgBase) SetID(id string) {
	panic("unsupported")
}

func (m *UnsupportedPwMsgBase) SetType(t string) {
	panic("unsupported")
}

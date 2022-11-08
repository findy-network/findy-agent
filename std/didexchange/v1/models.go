package v1

import (
	our "github.com/findy-network/findy-agent/std/decorator"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/protocol/decorator"
)

type Request struct {
	Type   string                `json:"@type,omitempty"`
	ID     string                `json:"@id,omitempty"`
	DID    string                `json:"did,omitempty"`
	DIDDoc *decorator.Attachment `json:"did_doc~attach,omitempty"`
	Thread *our.Thread           `json:"~thread,omitempty"`
	Label  string                `json:"label,omitempty"`
}

type Response struct {
	Type   string                `json:"@type,omitempty"`
	ID     string                `json:"@id,omitempty"`
	DID    string                `json:"did,omitempty"`
	DIDDoc *decorator.Attachment `json:"did_doc~attach,omitempty"`
	Thread *our.Thread           `json:"~thread,omitempty"`
}

type Complete struct {
	Type   string      `json:"@type,omitempty"`
	ID     string      `json:"@id,omitempty"`
	Thread *our.Thread `json:"~thread,omitempty"`
}

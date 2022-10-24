package didexchange1

import (
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/protocol/decorator"
)

type Request struct {
	Type   string                `json:"@type,omitempty"`
	ID     string                `json:"@id,omitempty"`
	DID    string                `json:"did,omitempty"`
	DIDDoc *decorator.Attachment `json:"did_doc~attach,omitempty"`
	Thread *decorator.Thread     `json:"~thread,omitempty"`
	Label  string                `json:"label,omitempty"`
}

type Response struct {
	Type   string                `json:"@type,omitempty"`
	ID     string                `json:"@id,omitempty"`
	DID    string                `json:"did,omitempty"`
	DIDDoc *decorator.Attachment `json:"did_doc~attach,omitempty"`
	Thread *decorator.Thread     `json:"~thread,omitempty"`
}

type Complete struct {
	Type   string            `json:"@type,omitempty"`
	ID     string            `json:"@id,omitempty"`
	Thread *decorator.Thread `json:"~thread,omitempty"`
}

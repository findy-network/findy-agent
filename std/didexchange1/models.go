package didexchange1

import (
	"github.com/findy-network/findy-agent/std/decorator"
)

type Request struct {
	Type   string                 `json:"@type,omitempty"`
	ID     string                 `json:"@id,omitempty"`
	DID    string                 `json:"did,omitempty"`
	DIDDoc []decorator.Attachment `json:"did_doc~attach,omitempty"`
	Thread *decorator.Thread      `json:"~thread,omitempty"`
	Label  string                 `json:"label,omitempty"`
}

type Response struct {
	Type   string                 `json:"@type,omitempty"`
	ID     string                 `json:"@id,omitempty"`
	DID    string                 `json:"did,omitempty"`
	DIDDoc []decorator.Attachment `json:"did_doc~attach,omitempty"`
	Thread *decorator.Thread      `json:"~thread,omitempty"`
}

type Complete struct {
	Type   string            `json:"@type,omitempty"`
	ID     string            `json:"@id,omitempty"`
	Thread *decorator.Thread `json:"~thread,omitempty"`
}

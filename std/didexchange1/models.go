package didexchange1

import (
	"github.com/findy-network/findy-agent/std/decorator"
)

type Request struct {
	Type   string                 `json:"@type,omitempty"`
	ID     string                 `json:"@id,omitempty"`
	Label  string                 `json:"label,omitempty"`
	DID    string                 `json:"did,omitempty"`
	DIDDoc []decorator.Attachment `json:"did_doc~attach,omitempty"`
	Thread *decorator.Thread      `json:"~thread,omitempty"`
}

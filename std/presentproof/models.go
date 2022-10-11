package presentproof

import "github.com/findy-network/findy-agent/std/decorator"

// MARK: Request

type Request struct {
	Type                 string                 `json:"@type,omitempty"`
	ID                   string                 `json:"@id,omitempty"`
	Comment              string                 `json:"comment,omitempty"`
	RequestPresentations []decorator.Attachment `json:"request_presentations~attach,omitempty"`
	Thread               *decorator.Thread      `json:"~thread,omitempty"`
}

// MARK: Presentation

type Presentation struct {
	Type                 string                 `json:"@type,omitempty"`
	ID                   string                 `json:"@id,omitempty"`
	Comment              string                 `json:"comment,omitempty"`
	PresentationAttaches []decorator.Attachment `json:"presentations~attach,omitempty"`
	Thread               *decorator.Thread      `json:"~thread,omitempty"`
}

// MARK: Propose

type Propose struct {
	Type                 string            `json:"@type,omitempty"`
	ID                   string            `json:"@id,omitempty"`
	Comment              string            `json:"comment,omitempty"`
	PresentationProposal *Preview          `json:"presentation_proposal,omitempty"`
	Thread               *decorator.Thread `json:"~thread,omitempty"`
}

// MARK: Preview

type Preview struct {
	Type       string      `json:"@type,omitempty"`
	Attributes []Attribute `json:"attributes"`
	Predicates []Predicate `json:"predicates"`
}

type Attribute struct {
	Name      string `json:"name"`
	CredDefID string `json:"cred_def_id,omitempty"`

	// https://github.com/hyperledger/aries-rfcs/blob/master/features/0037-present-proof/README.md#mime-type-and-value
	MimeType string `json:"mime_type,omitempty"`
	Value    string `json:"value,omitempty"`

	// https://github.com/hyperledger/aries-rfcs/blob/master/features/0037-present-proof/README.md#referent
	Referent string `json:"referent,omitempty"`
}

// Predicate is definition type of Preview struct.
//
//	https://github.com/hyperledger/aries-rfcs/blob/master/features/0037-present-proof/README.md#predicates
type Predicate struct {
	Name      string `json:"name"`
	CredDefID string `json:"cred_def_id"`
	Predicate string `json:"predicate"` // "<", "<=", ">=", ">"
	Threshold string `json:"threshold"`
}

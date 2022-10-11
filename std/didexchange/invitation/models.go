// Taken from aries-framework-go, and heavily modified. The idea is to replace
// these with the aries-framework-go when it's ready. Until now we use our own
// minimalistic solution.

// Package invitation is for invitation data model. It includes the JSON struct.
package invitation

import (
	"github.com/findy-network/findy-agent/std/decorator"
)

// Invitation model
//
// Invitation defines DID exchange invitation message
// https://github.com/hyperledger/aries-rfcs/tree/master/features/0023-did-exchange#0-invitation-to-exchange
type Invitation struct {
	// the Image URL of the connection invitation
	ImageURL string `json:"imageUrl,omitempty"`

	// the Service endpoint of the connection invitation
	ServiceEndpoint string `json:"serviceEndpoint,omitempty"`

	// the RecipientKeys for the connection invitation
	RecipientKeys []string `json:"recipientKeys,omitempty"`

	// the ID of the connection invitation
	ID string `json:"@id,omitempty"`

	// the Label of the connection invitation
	Label string `json:"label,omitempty"`

	// the DID of the connection invitation
	DID string `json:"did,omitempty"`

	// the RoutingKeys of the connection invitation
	RoutingKeys []string `json:"routingKeys,omitempty"`

	// the Type of the connection invitation
	Type   string            `json:"@type,omitempty"`
	Thread *decorator.Thread `json:"~thread,omitempty"`
}

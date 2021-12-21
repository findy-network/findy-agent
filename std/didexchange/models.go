// Taken from aries-framework-go, and heavily modified. The idea is to replace
// these with the aries-framework-go when it's ready. Until now we use our own
// minimalistic solution.

// Package didexchange is currently used for connection protocol implementation
package didexchange

import (
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-agent/std/did"
	"github.com/golang/glog"
	"github.com/lainio/err2/assert"
)

// Request defines a2a DID exchange request
// https://github.com/hyperledger/aries-rfcs/tree/master/features/0023-did-exchange#1-exchange-request
type Request struct {
	Type       string            `json:"@type,omitempty"`
	ID         string            `json:"@id,omitempty"`
	Label      string            `json:"label,omitempty"`
	Connection *Connection       `json:"connection,omitempty"`
	Thread     *decorator.Thread `json:"~thread,omitempty"`
}

// Response defines a2a DID exchange response
// https://github.com/hyperledger/aries-rfcs/tree/master/features/0023-did-exchange#2-exchange-response
type Response struct {
	Type                string               `json:"@type,omitempty"`
	ID                  string               `json:"@id,omitempty"`
	ConnectionSignature *ConnectionSignature `json:"connection~sig,omitempty"`
	Thread              *decorator.Thread    `json:"~thread,omitempty"`

	Connection *Connection `json:"-"` // Actual data, to be signed or verified
}

// ConnectionSignature connection signature
type ConnectionSignature struct {
	Type       string `json:"@type,omitempty"`
	Signature  string `json:"signature,omitempty"`
	SignedData string `json:"sig_data,omitempty"`
	SignVerKey string `json:"signer,omitempty"` // Todo: was signers
}

// Connection connection
type Connection struct {
	DID    string   `json:"DID,omitempty"`
	DIDDoc *did.Doc `json:"DIDDoc,omitempty"` // todo: was did_doc
}

func RouteForConnection(conn *Connection) (route []string) {
	// Find the routing keys from the request
	if conn == nil {
		glog.Warningln("RouteForConnection - no DIDExchange request found")
		return
	}
	if conn.DIDDoc == nil {
		glog.Warningln("RouteForConnection - request does not contain DIDDoc")
		return
	}
	assert.D.True(len(conn.DIDDoc.Service) > 0)
	route = conn.DIDDoc.Service[0].RoutingKeys
	return
}

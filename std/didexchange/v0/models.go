// Taken from aries-framework-go, and heavily modified. The idea is to replace
// these with the aries-framework-go when it's ready. Until now we use our own
// minimalistic solution.

// Package didexchange is currently used for connection protocol implementation
package v0

import (
	"encoding/json"

	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-agent/std/decorator"
	sov "github.com/findy-network/findy-agent/std/sov/did"
	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
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

// Connection is a connection definition
type Connection struct {
	DID    string
	DIDDoc core.DIDDoc // handles both types of DIDDoc: sov and AFGO
}

type AFGOConnection struct {
	DID string   `json:"DID,omitempty"`
	Doc *did.Doc `json:"DIDDoc,omitempty"`
}

type DataConnection struct {
	DID string   `json:"DID,omitempty"`
	Doc *sov.Doc `json:"DIDDoc,omitempty"`
}

func (c *Connection) MarshalJSON() (_ []byte, err error) {
	defer err2.Handle(&err, "marshal connection")

	switch doc := c.DIDDoc.(type) {
	case *did.Doc:
		out := AFGOConnection{
			DID: c.DID,
			Doc: doc,
		}
		return json.Marshal(out)
	case *sov.Doc:
		out := DataConnection{
			DID: c.DID,
			Doc: doc,
		}
		return json.Marshal(out)
	default:
		assert.NotImplemented()
		return nil, nil
	}
}

func (c *Connection) UnmarshalJSON(b []byte) (err error) {
	defer err2.Handle(&err)

	data := new(AFGOConnection)
	if err := json.Unmarshal(b, data); err == nil {
		isNotIndyAgent := len(data.Doc.Service) > 0 && data.Doc.Service[0].Type != "IndyAgent"
		if isNotIndyAgent {
			c.DID = data.DID
			c.DIDDoc = data.Doc
			return nil
		}
	}

	dataSov := new(DataConnection)
	try.To(json.Unmarshal(b, dataSov))
	c.DID = dataSov.DID
	c.DIDDoc = dataSov.Doc

	return nil
}
